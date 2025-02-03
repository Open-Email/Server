package main

import (
	"email.mercata.com/internal/email/storage"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"net/http"
	"strings"
)

func (app *application) listLinks(w http.ResponseWriter, r *http.Request) {
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

	linksLines, err := storage.ListLinks(userHomeDirPath)
	if err != nil {
		app.serverError(w, err)
		return
	}
	for _, line := range linksLines {
		_, err = fmt.Fprintln(w, line)
		if err != nil {
			app.serverError(w, err)
			return
		}
	}
}

func (app *application) storeLink(w http.ResponseWriter, r *http.Request) {
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

	params := httprouter.ParamsFromContext(r.Context())
	link := strings.ToLower(params.ByName("link"))

	contact, err := ioutil.ReadAll(r.Body)
	if err != nil {
		app.serverError(w, err)
		return
	}

	err = storage.StoreLink(userHomeDirPath, link, contact)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (app *application) deleteLink(w http.ResponseWriter, r *http.Request) {
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

	params := httprouter.ParamsFromContext(r.Context())
	link := strings.ToLower(params.ByName("link"))

	err = storage.DeleteLink(userHomeDirPath, link)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
