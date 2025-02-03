package main

import (
	"email.mercata.com/internal/email/notification"
	"fmt"
	"net/http"
)

func (app *application) getNotifications(w http.ResponseWriter, r *http.Request) {
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

	lines, err := notification.ListAll(userHomeDirPath)
	if err != nil {
		app.serverError(w, err)
		return
	}

	for _, line := range lines {
		_, err = fmt.Fprintln(w, line)
		if err != nil {
			app.serverError(w, err)
			return
		}
	}
}
