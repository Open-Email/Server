package main

import (
	"email.mercata.com/internal/consts"
	"email.mercata.com/internal/email/profile"
	"email.mercata.com/internal/utils"
	"errors"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

/* Profile image, standard format of 400x400 (recommendation). Large images
 * by dimension and size should be ignored by email clients.
 *
 * The image is not required. If not present, clients should handle HTTP 404
 * response and show a static placeholder.
 *
 * Once the image is retrieved, its HTTP headers may be used to verify authenticity
 * after which it can be cached locally as well resized in required dimensions,
 * such as thumbnails. This responsibility is passed on to clients.
 */
func (app *application) getProfile(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	domain := strings.ToLower(params.ByName("domain"))
	user := strings.ToLower(params.ByName("user"))

	userHomeDir, homeDirExists, err := app.userHomePath(domain, user)
	if err != nil {
		app.serverError(w, err)
		return
	}
	if !homeDirExists {
		app.notFound(w)
		return
	}

	app.serveProfileFile(w, r, profile.GetLocalProfileDataPath(userHomeDir), "text/plain; charset=utf-8")
}

func (app *application) getProfileImage(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	domain := strings.ToLower(params.ByName("domain"))
	user := strings.ToLower(params.ByName("user"))

	userHomeDir, homeDirExists, err := app.userHomePath(domain, user)
	if err != nil {
		app.serverError(w, err)
		return
	}
	if !homeDirExists {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	imagePath := profile.GetLocalProfileImagePath(userHomeDir)
	imageExists, err := utils.FilePathExists(imagePath)
	if err != nil {
		app.serverError(w, err)
		return
	}
	if !imageExists {
		app.notFound(w)
		return
	}

	mimeType, imateTypePermitted, err := profile.ImagePathFileTypeIsPermitted(imagePath)
	if err != nil {
		app.serverError(w, err)
		return
	}
	if !imateTypePermitted {
		app.infoLog.Printf("Not serving unpermitted profile image type '%s' for %s@%s", *mimeType, user, domain)
		app.notFound(w)
		return
	}

	app.serveProfileFile(w, r, imagePath, *mimeType)
}

func (app *application) serveProfileFile(w http.ResponseWriter, r *http.Request, filePath string, contentTypeOverride string) {
	fstat, err := os.Stat(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			app.notFound(w)
			return
		}
		app.serverError(w, err)
		return
	}

	if modifiedSince, err := time.Parse(http.TimeFormat, r.Header.Get("If-Modified-Since")); err == nil && fstat.ModTime().Before(modifiedSince.Add(time.Second)) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", contentTypeOverride)
	w.Header().Set("Last-Modified", fstat.ModTime().UTC().Format(http.TimeFormat))
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", consts.MAX_CACHE_DURATION))
	w.Header().Set("Expires", time.Now().Add(consts.MAX_CACHE_DURATION*time.Second).UTC().Format(http.TimeFormat))

	_, err = w.Write(data)
	if err != nil {
		app.serverError(w, err)
		return
	}
}
