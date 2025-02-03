package main

import (
	"email.mercata.com/internal/consts"
	cryptoPkg "email.mercata.com/internal/crypto"
	"email.mercata.com/internal/email/notification"
	profilePkg "email.mercata.com/internal/email/profile"
	"email.mercata.com/internal/email/storage"
	"email.mercata.com/internal/utils"
	"net/http"
	"strings"
)

// When notification is written for the same system, the server does not distinguish
// between local and remote user. If a notification caller is also a local user, the
// system SHOULD NOT be aware of it, otherwise the caller's email address is revealed.
func (app *application) writeNotification(w http.ResponseWriter, r *http.Request) {
	notifierAttrs := utils.ParseHeadersAttributes(r.Header.Get(consts.NOTIFICATION_ORIGIN_HEADER))

	algorithm, ok := notifierAttrs["algorithm"]
	if !ok {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	originEncryptedEmailAddress, ok := notifierAttrs["value"]
	if !ok {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	encryptionKeyFingerprint, ok := notifierAttrs["key"]
	if !ok {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	domain, ok := r.Context().Value(domainContextKey).(string)
	if !ok {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	user, ok := r.Context().Value(userContextKey).(string)
	if !ok {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	link, ok := r.Context().Value(linkContextKey).(string)
	if !ok {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	userHomeDirPath, homeDirExists, err := app.userHomePath(domain, user)
	if err != nil {
		app.serverError(w, err)
		return
	}
	if !homeDirExists {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	profile, err := profilePkg.GetLocalProfile(userHomeDirPath, domain, user)
	if err != nil {
		app.serverError(w, err)
		return
	}

	linkExists, err := storage.UserHasLink(userHomeDirPath, link)
	if err != nil {
		app.serverError(w, err)
		return
	}

	if !profile.PublicAccess {
		if !linkExists {
			app.clientError(w, http.StatusForbidden)
			return
		}
	}

	// Algorithm for encrypting the origin email address must match
	// the reader's profile encryption algorithm. This server uses only one by default.
	if cryptoPkg.ANONYMOUS_ENCRYPTION_CIPHER != strings.ToLower(algorithm) {
		app.clientError(w, http.StatusBadRequest)
		return
	}
	// The fingerprint must match at the time of making the notifications
	if profile.User.PublicEncryptionKeyFingerprint != encryptionKeyFingerprint {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	notifierKeyFingerprint, ok := r.Context().Value(signingFingerprintContextKey).(string)
	if !ok {
		app.serverError(w, err)
		return
	}

	err = notification.Store(userHomeDirPath, link, string(originEncryptedEmailAddress), notifierKeyFingerprint, profile.User.PublicEncryptionKeyFingerprint)
	if err != nil {
		app.serverError(w, err)
		return
	}

	if profile.IsAway {
		// The StatusAccepted should indicate the Away status to clients if they have not checked previously.
		w.WriteHeader(http.StatusAccepted)
		return
	}

	w.WriteHeader(http.StatusOK)
}
