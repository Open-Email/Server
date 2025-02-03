package main

import (
	"bufio"
	"bytes"
	"email.mercata.com/internal/consts"
	"email.mercata.com/internal/email/message"
	"email.mercata.com/internal/email/storage"
	userpkg "email.mercata.com/internal/email/user"
	"email.mercata.com/internal/utils"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (app *application) listBroadcastMessages(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	domain := strings.ToLower(params.ByName("domain"))
	user := strings.ToLower(params.ByName("user"))

	if domain == "" || user == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	stream := strings.ToLower(params.ByName("stream"))
	userHomeDirPath := filepath.Join(app.config.dataDirPath, domain, user)

	w.Header().Set("Content-Type", "text/plain")

	messageIDs, err := storage.FilterMessagesIndex(userHomeDirPath, "", "", stream)
	if err != nil {
		app.serverError(w, err)
		return
	}
	for _, messageID := range messageIDs {
		_, messageExists, err := storage.MessageExists(userHomeDirPath, messageID)
		if err != nil {
			app.serverError(w, err)
			return
		}
		if !messageExists {
			continue
		}
		_, err = fmt.Fprintln(w, messageID)
		if err != nil {
			app.serverError(w, err)
			return
		}
	}
}

func (app *application) getBroadcastMessage(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	domain := strings.ToLower(params.ByName("domain"))
	user := strings.ToLower(params.ByName("user"))
	messageID := strings.ToLower(params.ByName("messageid"))

	if domain == "" || user == "" || messageID == "" {
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

	_, messageExists, err := storage.MessageExists(userHomeDirPath, messageID)
	if err != nil {
		app.serverError(w, err)
		return
	}
	if !messageExists {
		app.notFound(w)
		return
	}

	envelopeFileContents, err := ioutil.ReadFile(storage.MessageEnvelopePath(userHomeDirPath, messageID))
	if err != nil {
		app.serverError(w, err)
		return
	}

	// The authenticated API must be used for access to private messages.
	// This scenario is very unlikely, as the user had to user authenticated
	// API to fetch the messageID.
	if bytes.Contains(envelopeFileContents, []byte(message.HEADER_MESSAGE_ACCESS)) {
		app.notFound(w)
		return
	}

	err = writeEnvelopeAsResponseHeaders(&envelopeFileContents, w)
	if err != nil {
		app.serverError(w, err)
		return
	}

	payloadFile, err := os.Open(storage.MessagePayloadPath(userHomeDirPath, messageID))
	if err != nil {
		app.serverError(w, err)
		return
	}
	defer payloadFile.Close()
	payloadFileStat, err := payloadFile.Stat()
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+consts.MESSAGE_DIR_PAYLOAD_FILE_NAME)
	w.Header().Set("Content-Length", strconv.FormatInt(payloadFileStat.Size(), 10))
	http.ServeContent(w, r, consts.MESSAGE_DIR_PAYLOAD_FILE_NAME, payloadFileStat.ModTime(), payloadFile)
}

// Private (Link) Messages
func (app *application) listLinkMessages(w http.ResponseWriter, r *http.Request) {
	publicKeyFingerprint, ok := r.Context().Value(signingFingerprintContextKey).(string)
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

	params := httprouter.ParamsFromContext(r.Context())
	stream := strings.ToLower(params.ByName("stream"))

	userHomeDirPath := filepath.Join(app.config.dataDirPath, domain, user)

	w.Header().Set("Content-Type", "text/plain")

	messageIDs, err := storage.FilterMessagesIndex(userHomeDirPath, link, publicKeyFingerprint, stream)
	if err != nil {
		app.serverError(w, err)
		return
	}

	for _, messageID := range messageIDs {
		_, err = fmt.Fprintln(w, messageID)
		if err != nil {
			app.serverError(w, err)
			return
		}
	}
}

func (app *application) getLinkMessage(w http.ResponseWriter, r *http.Request) {
	publicKeyFingerprint, ok := r.Context().Value(signingFingerprintContextKey).(string)
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

	params := httprouter.ParamsFromContext(r.Context())
	messageID := strings.ToLower(params.ByName("messageid"))

	userHomeDirPath, homeDirExists, err := app.userHomePath(domain, user)
	if err != nil {
		app.serverError(w, err)
		return
	}
	if !homeDirExists {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	_, messageExists, err := storage.MessageExists(userHomeDirPath, messageID)
	if err != nil {
		app.serverError(w, err)
		return
	}
	if !messageExists {
		app.notFound(w)
		return
	}

	envelopeFileContents, err := ioutil.ReadFile(storage.MessageEnvelopePath(userHomeDirPath, messageID))
	if err != nil {
		app.serverError(w, err)
		return
	}

	authorized, err := message.LinkFingerprintExistsInAccessList(link, publicKeyFingerprint, &envelopeFileContents)
	if err != nil {
		app.serverError(w, err)
		return
	}

	if !authorized {
		app.clientError(w, http.StatusUnauthorized)
		return
	}

	err = writeEnvelopeAsResponseHeaders(&envelopeFileContents, w)
	if err != nil {
		app.serverError(w, err)
		return
	}

	payloadFile, err := os.Open(storage.MessagePayloadPath(userHomeDirPath, messageID))
	if err != nil {
		app.serverError(w, err)
		return
	}
	defer payloadFile.Close()
	payloadFileStat, err := payloadFile.Stat()
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+consts.MESSAGE_DIR_PAYLOAD_FILE_NAME)
	w.Header().Set("Content-Length", strconv.FormatInt(payloadFileStat.Size(), 10))
	http.ServeContent(w, r, consts.MESSAGE_DIR_PAYLOAD_FILE_NAME, payloadFileStat.ModTime(), payloadFile)

	// Don't log own access
	if userpkg.SelfLink(user, domain) != link {
		err = storage.LogMessageAccess(userHomeDirPath, messageID, link)
		if err != nil {
			app.serverError(w, err)
			return
		}
	}
}

func writeEnvelopeAsResponseHeaders(envelopeContent *[]byte, w http.ResponseWriter) error {
	scanner := bufio.NewScanner(bytes.NewReader(*envelopeContent))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line[0] == '#' {
			continue
		}
		parts := strings.SplitN(line, message.HEADER_KEY_VALUE_SEPARATOR, 2)
		if len(parts) != 2 {
			continue
		}
		if utils.ListContains(message.PERMITTED_ENVELOPE_KEYS, strings.ToLower(parts[0])) {
			w.Header().Set(parts[0], parts[1])
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}
