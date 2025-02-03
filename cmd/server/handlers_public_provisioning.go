package main

import (
	"email.mercata.com/internal/consts"
	"email.mercata.com/internal/email/profile"
	"email.mercata.com/internal/nonce"
	"email.mercata.com/internal/utils"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"net/http"
	"strings"
)

// TODO Limit to number of request per IP to one per hour?

// create directory and store minimal profile with public encryption and public signing key given in request

func (app *application) provisionUser(w http.ResponseWriter, r *http.Request) {
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

	if domain == "" || user == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	if !utils.ListContains(app.config.provisioning.domains, domain) {
		app.forbidden(w)
		return
	}

	userHomeDir, homeDirExists, err := app.userHomePath(domain, user)
	if err != nil {
		app.serverError(w, err)
		return
	}
	if homeDirExists {
		app.clientError(w, http.StatusConflict)
		return
	}

	// Provision account now

	profData, err := ioutil.ReadAll(r.Body)
	if err != nil {
		app.serverError(w, err)
		return
	}

	p := profile.Profile{}
	err = profile.ParseProfile(&p, profData)
	if err != nil || !profile.IsFunctionalProfile(&p) {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// The nonce signing key must be present and match the key in profile data
	if p.User.PublicSigningKeyBase64 != "" && p.User.PublicSigningKeyBase64 != n.SigningKeyBase64 {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	err = profile.SetLocalProfile(userHomeDir, &profData)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
