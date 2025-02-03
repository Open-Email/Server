package main

import (
	"bytes"
	"email.mercata.com/internal/utils"
	"errors"
	"fmt"
	"github.com/go-playground/form/v4"
	"github.com/justinas/nosurf"
	"net/http"
	"path/filepath"
	"runtime/debug"
	"time"
)

func (app *application) serverError(w http.ResponseWriter, err error) {
	trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
	app.errorLog.Output(2, trace)

	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (app *application) clientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

func (app *application) notFound(w http.ResponseWriter) {
	app.clientError(w, http.StatusNotFound)
}

func (app *application) forbidden(w http.ResponseWriter) {
	app.clientError(w, http.StatusForbidden)
}

func (app *application) render(w http.ResponseWriter, status int, page string, data *templateData) {
	ts, ok := app.templateCache[page]
	if !ok {
		err := fmt.Errorf("the template %s does not exist", page)
		app.serverError(w, err)
		return
	}

	buf := new(bytes.Buffer)

	err := ts.ExecuteTemplate(buf, "base", data)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.WriteHeader(status)
	buf.WriteTo(w)
}

func (app *application) newTemplateData(r *http.Request) *templateData {
	randomNonce, ok := r.Context().Value(randomNonceContextKey).(string)
	if !ok {
		randomNonce = ""
	}

	return &templateData{
		RandomNonce: randomNonce,
		CurrentYear: time.Now().Year(),
		CSRFToken:   nosurf.Token(r),
		CurrentPath: r.URL.Path,
	}
}

func (app *application) decodePostForm(r *http.Request, dst any) error {
	err := r.ParseForm()
	if err != nil {
		return err
	}

	err = app.formDecoder.Decode(dst, r.PostForm)
	if err != nil {
		var invalidDecoderError *form.InvalidDecoderError

		if errors.As(err, &invalidDecoderError) {
			panic(err)
		}
		return err
	}

	return nil
}

func (app *application) domainHomePathExists(domain string) (bool, error) {
	return utils.FilePathExists(filepath.Join(app.config.dataDirPath, domain))
}

func (app *application) userHomePath(domain, user string) (string, bool, error) {
	homePath := filepath.Join(app.config.dataDirPath, domain, user)
	exists, err := utils.FilePathExists(homePath)
	return homePath, exists, err
}
