package main

import (
	"email.mercata.com/internal/consts"
	messagePkg "email.mercata.com/internal/email/message"
	"email.mercata.com/internal/email/storage"
	utils "email.mercata.com/internal/utils"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const MESSAGE_ID_ROUTER_PARAM = "mid"

func (app *application) getMessagesStatus(w http.ResponseWriter, r *http.Request) {
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

	userHomeDirPath := filepath.Join(app.config.dataDirPath, domain, user)

	w.Header().Set("Content-Type", "text/plain")

	msgRow, err := storage.MessagesStatus(userHomeDirPath)
	if err != nil {
		app.serverError(w, err)
		return
	}

	for _, line := range msgRow {
		_, err = fmt.Fprintln(w, line)
		if err != nil {
			app.serverError(w, err)
			return
		}
	}
}

func (app *application) storeMessage(w http.ResponseWriter, r *http.Request) {
	maxAllowedSize := int64(consts.DEFAULT_MAX_CONTENT_SIZE)

	contentLength := maxAllowedSize
	contentLengthStr := r.Header.Get("Content-Length")
	if contentLengthStr == "" {
		app.infoLog.Printf("Content-Length not provided, assuming maximum allowed size: %d", consts.DEFAULT_MAX_CONTENT_SIZE)
	} else {
		var err error
		contentLength, err = strconv.ParseInt(contentLengthStr, 10, 64)
		if err != nil {
			contentLength = maxAllowedSize
			app.errorLog.Printf("Invalid Content-Length provided [%s], assuming maximum allowed size %d", contentLengthStr, maxAllowedSize)
		} else {
			if contentLength > maxAllowedSize {
				http.Error(w, "Unacceptable Message-Size", http.StatusBadRequest)
				return
			}
		}
	}

	limitedReader := io.LimitReader(r.Body, contentLength)

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

	// Maybe the user tries to fill up our server
	homeDirSize, err := utils.DirectorySize(userHomeDirPath)
	if err != nil {
		app.serverError(w, err)
		return
	}
	if (homeDirSize + contentLength) > consts.MAX_HOME_DIR_SIZE {
		app.infoLog.Printf("home directory too large %d, not accepting additional %d", homeDirSize, contentLength)
		app.clientError(w, http.StatusRequestEntityTooLarge)
		return
	}

	// We need the message ID from the envelope
	message, err := messagePkg.MessageFromHeadersData(r.Header)
	if err != nil {
		app.errorLog.Printf("failed to parse request headers %s", err)
		app.serverError(w, err)
		return
	}

	_, messageExists, err := storage.MessageExists(userHomeDirPath, message.ID)
	if err != nil {
		app.serverError(w, err)
		return
	}
	if messageExists {
		app.clientError(w, http.StatusConflict)
		return
	}

	messagePath, err := storage.CreateMessageDir(userHomeDirPath, message.ID)
	if err != nil {
		app.errorLog.Printf("failed to create message path: %s", err)
		app.serverError(w, err)
		return
	}
	envelopePath := storage.MessageEnvelopePath(userHomeDirPath, message.ID)
	payloadPath := storage.MessagePayloadPath(userHomeDirPath, message.ID)

	payloadFile, err := os.Create(payloadPath)
	if err != nil {
		app.errorLog.Printf("failed to create body file: %s", err)
		// DRY-UP!
		err := storage.DeleteMessageDir(userHomeDirPath, message.ID)
		if err != nil {
			app.errorLog.Printf("FATAL: failed to remove message path [%s]:", messagePath, err)
		}

		app.serverError(w, err)
		return
	}

	defer payloadFile.Close()

	buffer := make([]byte, 64*1024)
	_, err = io.CopyBuffer(payloadFile, limitedReader, buffer)

	if err != nil {
		app.errorLog.Printf("failed to save request body: %s", err)
		// DRY-UP!
		err := storage.DeleteMessageDir(userHomeDirPath, message.ID)
		if err != nil {
			app.errorLog.Printf("FATAL: failed to remove message path [%s]:", messagePath, err)
		}
		app.serverError(w, err)
		return
	}
	defer r.Body.Close()

	envelopeDumpStr := []byte(strings.Join(message.EnvelopeHeadersList, "\n"))
	err = ioutil.WriteFile(envelopePath, append(envelopeDumpStr, '\n'), 0644)
	if err != nil {
		app.errorLog.Printf("failed to save request envelope: %s", err)
		// DRY-UP!
		err := storage.DeleteMessageDir(userHomeDirPath, message.ID)
		if err != nil {
			app.errorLog.Printf("FATAL: failed to remove message path [%s]:", messagePath, err)
		}
		app.serverError(w, err)
		return
	}

	for _, reader := range message.Readers {
		err := storage.WriteMessageIndex(userHomeDirPath, reader.Link, reader.User.PublicSigningKeyFingerprint, message.StreamID, message.ID)
		if err != nil {
			app.errorLog.Printf("FATAL: failed to write message index: %s", err)
			// DRY-UP!
			err := storage.DeleteMessageDir(userHomeDirPath, message.ID)
			if err != nil {
				app.errorLog.Printf("FATAL: failed to remove message path [%s]:", messagePath, err)
			}
			app.serverError(w, err)
			return
		}
		app.infoLog.Printf("added message reader %s for message %s", reader.Link, message.ID)
	}
	app.infoLog.Printf("message stored in %s", messagePath)
}

func (app *application) deleteMessage(w http.ResponseWriter, r *http.Request) {
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
	messageID := strings.ToLower(params.ByName(MESSAGE_ID_ROUTER_PARAM))
	if messageID == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	messagePath, messageExists, err := storage.MessageExists(userHomeDirPath, messageID)
	if err != nil {
		app.serverError(w, err)
		return
	}
	if !messageExists {
		app.notFound(w)
		return
	}
	err = storage.DeleteMessageDir(userHomeDirPath, messageID)
	if err != nil {
		app.errorLog.Printf("FATAL: failed to remove message path [%s]:", messagePath, err)
		app.serverError(w, err)
		return
	}
	app.infoLog.Printf("Removed message path [%s]:", messagePath)

	err = storage.RemoveMessageFromIndex(userHomeDirPath, messageID)
	if err != nil {
		app.errorLog.Printf("failed to remove message from index [%s]:", messagePath, err)
		// Not fatal as the cleanup will remove stale entries
	}
	w.WriteHeader(http.StatusOK)
}
