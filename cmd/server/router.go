package main

import (
	"email.mercata.com/internal/consts"
	"email.mercata.com/ui"
	"fmt"
	"github.com/justinas/alice"
	"net/http"
	"strings"
)

func (app *application) routes() http.Handler {
	app.router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.notFound(w)
	})

	app.router.MethodNotAllowed = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.clientError(w, http.StatusMethodNotAllowed)
	})

	fileServer := http.FileServer(http.FS(ui.Files))

	app.router.Handler(http.MethodGet, "/static/*filepath",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			eTagVal := `"` + app.eTag + `"`
			w.Header().Set("Etag", eTagVal)
			w.Header().Set("Cache-Control", "public, max-age=3600")

			if match := r.Header.Get("If-None-Match"); match != "" {
				if strings.Contains(match, eTagVal) {
					w.WriteHeader(http.StatusNotModified)
					return
				}
			}

			fileServer.ServeHTTP(w, r)
		}))

	naked := alice.New()
	publiclyAuthenticated := naked.Append(app.authenticatePublic)
	privatelyAuthenticated := naked.Append(app.authenticatePrivate)
	// dynamic := naked.Append(noSurf)

	// [COMPLETE] Well-known file for those making a CNAME to this server
	// The TLS certificate for the CNAMEing domains must be provisioned separately.
	app.router.Handler(http.MethodGet, "/.well-known/mail.txt", naked.ThenFunc(app.getWellKnownFile))

	// [COMPLETE] Check if mail agent recognized the domain
	app.router.Handler(http.MethodHead, fmt.Sprintf("/%s/:domain", consts.PUBLIC_API_PATH_PREFIX), naked.ThenFunc(app.checkDomainDelegation))
	app.router.Handler(http.MethodGet, fmt.Sprintf("/%s/:domain", consts.PUBLIC_API_PATH_PREFIX), naked.ThenFunc(app.checkDomainDelegation))
	app.router.Handler(http.MethodHead, fmt.Sprintf("/%s/:domain/:user", consts.PUBLIC_API_PATH_PREFIX), naked.ThenFunc(app.checkUserDelegation))
	app.router.Handler(http.MethodGet, fmt.Sprintf("/%s/:domain/:user", consts.PUBLIC_API_PATH_PREFIX), naked.ThenFunc(app.checkUserDelegation))

	// [COMPLETE] Fetching information about contacts
	app.router.Handler(http.MethodGet, fmt.Sprintf("/%s/:domain/:user/profile", consts.PUBLIC_API_PATH_PREFIX), naked.ThenFunc(app.getProfile))
	app.router.Handler(http.MethodGet, fmt.Sprintf("/%s/:domain/:user/image", consts.PUBLIC_API_PATH_PREFIX), naked.ThenFunc(app.getProfileImage))

	// TODO: public messages indexing, how to support it best? Mentions? Can the messages be served as HTML? Is there need?

	// [COMPLETE] Fetching remote broadcast messages
	app.router.Handler(http.MethodGet, fmt.Sprintf("/%s/:domain/:user/messages", consts.PUBLIC_API_PATH_PREFIX), naked.ThenFunc(app.listBroadcastMessages))
	app.router.Handler(http.MethodGet, fmt.Sprintf("/%s/:domain/:user/streams/:stream/messages", consts.PUBLIC_API_PATH_PREFIX), naked.ThenFunc(app.listBroadcastMessages))
	// Individual broadcast message
	app.router.Handler(http.MethodGet, fmt.Sprintf("/%s/:domain/:user/messages/:messageid", consts.PUBLIC_API_PATH_PREFIX), naked.ThenFunc(app.getBroadcastMessage))

	// [COMPLETE] Fetching remote private messages
	app.router.Handler(http.MethodGet, fmt.Sprintf("/%s/:domain/:user/link/:link/messages", consts.PUBLIC_API_PATH_PREFIX), publiclyAuthenticated.ThenFunc(app.listLinkMessages))
	app.router.Handler(http.MethodGet, fmt.Sprintf("/%s/:domain/:user/link/:link/streams/:stream/messages", consts.PUBLIC_API_PATH_PREFIX), publiclyAuthenticated.ThenFunc(app.listLinkMessages))
	app.router.Handler(http.MethodGet, fmt.Sprintf("/%s/:domain/:user/link/:link/messages/:messageid", consts.PUBLIC_API_PATH_PREFIX), publiclyAuthenticated.ThenFunc(app.getLinkMessage))

	// [COMPLETE] Storing a remote notification
	app.router.Handler(http.MethodHead, fmt.Sprintf("/%s/:domain/:user/link/:link/notifications", consts.PUBLIC_API_PATH_PREFIX), publiclyAuthenticated.ThenFunc(app.writeNotification))

	// Private API, authenticated ----

	app.router.Handler(http.MethodHead, fmt.Sprintf("/%s/:domain/:user", consts.PRIVATE_API_PATH_PREFIX), privatelyAuthenticated.ThenFunc(app.queryAccessToProfile))

	// [COMPLETE] Fetching notifications for own account
	app.router.Handler(http.MethodGet, fmt.Sprintf("/%s/:domain/:user/notifications", consts.PRIVATE_API_PATH_PREFIX), privatelyAuthenticated.ThenFunc(app.getNotifications))

	// [COMPLETE] Managing own profile
	app.router.Handler(http.MethodPut, fmt.Sprintf("/%s/:domain/:user/profile", consts.PRIVATE_API_PATH_PREFIX), privatelyAuthenticated.ThenFunc(app.setProfile))
	app.router.Handler(http.MethodPut, fmt.Sprintf("/%s/:domain/:user/image", consts.PRIVATE_API_PATH_PREFIX), privatelyAuthenticated.ThenFunc(app.setProfileImage))

	// [COMPLETE] Links
	app.router.Handler(http.MethodGet, fmt.Sprintf("/%s/:domain/:user/links", consts.PRIVATE_API_PATH_PREFIX), privatelyAuthenticated.ThenFunc(app.listLinks))
	app.router.Handler(http.MethodPut, fmt.Sprintf("/%s/:domain/:user/links/:link", consts.PRIVATE_API_PATH_PREFIX), privatelyAuthenticated.ThenFunc(app.storeLink))
	app.router.Handler(http.MethodDelete, fmt.Sprintf("/%s/:domain/:user/links/:link", consts.PRIVATE_API_PATH_PREFIX), privatelyAuthenticated.ThenFunc(app.deleteLink))

	// [COMPLETE] Managing messages
	app.router.Handler(http.MethodGet, fmt.Sprintf("/%s/:domain/:user/messages", consts.PRIVATE_API_PATH_PREFIX), privatelyAuthenticated.ThenFunc(app.getMessagesStatus))
	app.router.Handler(http.MethodPost, fmt.Sprintf("/%s/:domain/:user/messages", consts.PRIVATE_API_PATH_PREFIX), privatelyAuthenticated.ThenFunc(app.storeMessage))
	app.router.Handler(http.MethodDelete, fmt.Sprintf("/%s/:domain/:user/messages/:mid", consts.PRIVATE_API_PATH_PREFIX), privatelyAuthenticated.ThenFunc(app.deleteMessage))

	if app.config.provisioning.enabled {
		// Provisioning API, public (if enabled)
		app.router.Handler(http.MethodPost, fmt.Sprintf("/%s/:domain/:user", consts.PRIVATE_PROVISION_PATH_PREFIX), naked.ThenFunc(app.provisionUser))
	}

	//app.secureHeaders
	standard := alice.New(app.recoverPanic, app.logRequest)

	return standard.Then(app.router)
}
