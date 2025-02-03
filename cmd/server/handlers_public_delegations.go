package main

import (
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"strings"
)

func (app *application) getWellKnownFile(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, app.config.mailAgentHostname)
}

func (app *application) checkDomainDelegation(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	domain := strings.ToLower(params.ByName("domain"))
	domainExists, err := app.domainHomePathExists(domain)
	if err != nil {
		app.serverError(w, err)
		return
	}
	if !domainExists {
		app.notFound(w)
		return
	}

	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}

	fmt.Fprintln(w, "OK")
}

func (app *application) checkUserDelegation(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	domain := strings.ToLower(params.ByName("domain"))
	localPart := strings.ToLower(params.ByName("user"))
	_, homeDirExists, err := app.userHomePath(domain, localPart)
	if err != nil {
		app.serverError(w, err)
		return
	}
	if !homeDirExists {
		app.notFound(w)
		return
	}

	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}

	fmt.Fprintln(w, "OK")
}
