package main

import (
	"context"
	"email.mercata.com/internal/consts"
	"email.mercata.com/internal/crypto"
	"email.mercata.com/internal/email/profile"
	"email.mercata.com/internal/nonce"
	"errors"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/nosurf"
	"net/http"
	"strings"
)

func noSurf(next http.Handler) http.Handler {
	csrfHandler := nosurf.New(next)
	csrfHandler.SetBaseCookie(http.Cookie{
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
	})

	return csrfHandler
}

func (app *application) secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inlineRandomNonce, err := crypto.GenerateRandomString(6)
		if err != nil {
			app.serverError(w, err)
			return
		}

		ctx := context.WithValue(r.Context(), randomNonceContextKey, inlineRandomNonce)
		r = r.WithContext(ctx)

		w.Header().Set("Content-Security-Policy",
			strings.Join([]string{
				"default-src 'self'",
				"style-src 'self'",
				fmt.Sprintf("script-src 'nonce-%s'", inlineRandomNonce),
			}, "; "))
		w.Header().Set("Referrer-Policy", "origin-when-cross-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "deny")
		w.Header().Set("X-XSS-Protection", "0")

		next.ServeHTTP(w, r)
	})
}

func (app *application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.infoLog.Printf("%s - %s %s %s", r.RemoteAddr, r.Proto, r.Method, r.URL.RequestURI())

		next.ServeHTTP(w, r)
	})
}

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverError(w, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *application) authenticatePrivate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n, err := nonce.FromHeader(r.Header.Get(consts.AUTHORIZATION_HEADER_NONCE))
		if err != nil {
			app.errorLog.Printf("Could not extract nonce. %s", err)
			app.clientError(w, http.StatusBadRequest)
			return
		}
		err = nonce.VerifySignature(n)
		if err != nil {
			app.errorLog.Printf("Could not verify nonce. %s", err)
			app.clientError(w, http.StatusBadRequest)
			return
		}

		params := httprouter.ParamsFromContext(r.Context())
		domain := strings.ToLower(params.ByName("domain"))
		user := strings.ToLower(params.ByName("user"))

		if domain == "" || user == "" {
			app.clientError(w, http.StatusBadRequest)
			return
		}

		userHomeDir, homeDirExists, err := app.userHomePath(domain, user)
		if err != nil {
			app.errorLog.Printf("Could not determine user home path: %s", err)
			app.serverError(w, err)
			return
		}
		if !homeDirExists {
			app.errorLog.Printf("No such user: %s/%s", domain, user)
			app.clientError(w, http.StatusUnauthorized)
			return
		}
		err = nonce.IsUnique(userHomeDir, n)
		if err != nil {
			if errors.Is(err, nonce.ErrorNonceReplay) {
				app.clientError(w, http.StatusUnauthorized)
				return
			}
			app.errorLog.Printf("Could not determine nonce uniqueness: %s", err)
			app.serverError(w, err)
			return
		}
		err = nonce.Record(userHomeDir, n)
		if err != nil {
			app.errorLog.Printf("Could not record nonce: %s", err)
			app.serverError(w, err)
			return
		}

		localProfile, err := profile.GetLocalProfile(userHomeDir, domain, user)
		if err != nil {
			app.errorLog.Printf("Could not load local profile: %s", err)
			app.notFound(w)
			return
		}

		if n.SigningKeyFingerprint != localProfile.User.PublicSigningKeyFingerprint &&
			((localProfile.User.PublicSigningKeyFingerprint != "") && n.SigningKeyFingerprint != localProfile.LastSigningKeyFingerprint) {
			app.clientError(w, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), domainContextKey, domain)
		ctx = context.WithValue(ctx, userContextKey, user)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

func (app *application) authenticatePublic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n, err := nonce.FromHeader(r.Header.Get(consts.AUTHORIZATION_HEADER_NONCE))
		if err != nil {
			app.errorLog.Printf("Could not extract nonce: %s", err)
			app.clientError(w, http.StatusBadRequest)
			return
		}
		err = nonce.VerifySignature(n)
		if err != nil {
			app.errorLog.Printf("Could not verify nonce: %s", err)
			app.clientError(w, http.StatusBadRequest)
			return
		}

		params := httprouter.ParamsFromContext(r.Context())
		domain := strings.ToLower(params.ByName("domain"))
		user := strings.ToLower(params.ByName("user"))
		link := strings.ToLower(params.ByName("link"))
		if domain == "" || user == "" || link == "" {
			app.clientError(w, http.StatusBadRequest)
			return
		}
		userHomeDir, homeDirExists, err := app.userHomePath(domain, user)
		if err != nil {
			app.serverError(w, err)
			return
		}
		if !homeDirExists {
			app.clientError(w, http.StatusBadRequest)
			return
		}

		err = nonce.IsUnique(userHomeDir, n)
		if err != nil {
			if errors.Is(err, nonce.ErrorNonceReplay) {
				app.clientError(w, http.StatusUnauthorized)
				return
			}
			app.serverError(w, err)
			return
		}
		err = nonce.Record(userHomeDir, n)
		if err != nil {
			app.serverError(w, err)
			return
		}

		// The fingerprint serves as proof of identity
		ctx := context.WithValue(r.Context(), signingFingerprintContextKey, n.SigningKeyFingerprint)
		ctx = context.WithValue(ctx, domainContextKey, domain)
		ctx = context.WithValue(ctx, userContextKey, user)
		ctx = context.WithValue(ctx, linkContextKey, link)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}
