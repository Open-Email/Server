package main

import (
	"email.mercata.com/internal/consts"
	addressPkg "email.mercata.com/internal/email/address"
	"email.mercata.com/internal/email/profile"
	"email.mercata.com/internal/utils"
	"io/ioutil"
	"net/http"
)

func (app *application) queryAccessToProfile(w http.ResponseWriter, r *http.Request) {
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

	_, homeDirExists, err := app.userHomePath(domain, user)
	if err != nil {
		app.serverError(w, err)
		return
	}
	if !homeDirExists {
		app.clientError(w, http.StatusBadRequest)
		return
	}
}

func (app *application) setProfile(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, consts.MAX_PROFILE_SIZE)

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

	userHomeDirPath, homeDirExists, err := app.userHomePath(domain, user)
	if err != nil {
		app.serverError(w, err)
		return
	}
	if !homeDirExists {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	profData, err := ioutil.ReadAll(r.Body)
	if err != nil {
		app.serverError(w, err)
		return
	}

	p := profile.Profile{RemoteBody: &profData}
	p.User.Address = addressPkg.JoinAddress(domain, user)
	p.User.LocalPart = user
	p.User.Domain = domain

	// Check for errors
	err = profile.ParseProfile(&p, profData)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	err = profile.SetLocalProfile(userHomeDirPath, &profData)
	if err != nil {
		app.serverError(w, err)
		return
	}
}

func (app *application) setProfileImage(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, consts.MAX_PROFILE_IMAGE_SIZE)

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

	userHomeDirPath, homeDirExists, err := app.userHomePath(domain, user)
	if err != nil {
		app.serverError(w, err)
		return
	}
	if !homeDirExists {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// TODO: Limit request size!
	profImageData, err := ioutil.ReadAll(r.Body)
	if err != nil {
		app.serverError(w, err)
		return
	}

	filetype, err := utils.DetermineFileTypeOfData(&profImageData)
	if err != nil {
		app.errorLog.Printf("could not determine image mimetype: %s", err)
		app.clientError(w, http.StatusBadRequest)
		return
	}
	if !profile.ImageMimeTypeIsPermitted(filetype) {
		app.errorLog.Printf("unpermitted image mimetype: %s", filetype)
		app.clientError(w, http.StatusBadRequest)
		return
	}

	err = profile.SetLocalProfileImage(userHomeDirPath, &profImageData)
	if err != nil {
		app.serverError(w, err)
		return
	}
}
