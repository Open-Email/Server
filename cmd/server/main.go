package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/go-playground/form/v4"
	"github.com/julienschmidt/httprouter"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"email.mercata.com/internal/crypto"
	"email.mercata.com/internal/utils"
)

const version = "1.0.0"

type config struct {
	port              int
	dataDirPath       string
	mailAgentHostname string

	provisioning struct {
		enabled bool
		domains []string
	}

	tls struct {
		enabled  bool
		certPath string
		keyPath  string
	}
}

type application struct {
	eTag          string
	randomNonce   string
	router        *httprouter.Router
	config        config
	errorLog      *log.Logger
	infoLog       *log.Logger
	templateCache map[string]*template.Template
	formDecoder   *form.Decoder
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "Server port")
	flag.StringVar(&cfg.mailAgentHostname, "agent-hostname", "", "Public agent hostname")
	flag.StringVar(&cfg.dataDirPath, "data-dir", "/tmp", "User data directory path")

	var provisioningDomainsStr string
	flag.StringVar(&provisioningDomainsStr, "provision", "", "Enable provisioning on listed comma separated domains")

	flag.BoolVar(&cfg.tls.enabled, "tls", false, "Enable TLS")
	flag.StringVar(&cfg.tls.certPath, "tls-cert", "./tls/cert.pem", "TLS Certificate path")
	flag.StringVar(&cfg.tls.keyPath, "tls-key", "./tls/key.pem", "TLS Key path")
	flag.Parse()

	if provisioningDomainsStr != "" {
		domainsStr := strings.Split(provisioningDomainsStr, ",")
		for _, domainHostname := range domainsStr {
			domainHostname = strings.ToLower(strings.TrimSpace(domainHostname))
			if utils.IsValidHostname(domainHostname) {
				cfg.provisioning.domains = append(cfg.provisioning.domains, domainHostname)
			}
			if len(cfg.provisioning.domains) > 0 {
				cfg.provisioning.enabled = true
			}
		}
	}

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	templateCache, err := newTemplateCache()
	if err != nil {
		errorLog.Fatal(err)
	}

	formDecoder := form.NewDecoder()

	app := &application{
		router:        httprouter.New(),
		config:        cfg,
		errorLog:      errorLog,
		infoLog:       infoLog,
		templateCache: templateCache,
		formDecoder:   formDecoder,
	}

	etag, err := crypto.GenerateRandomString(6)
	if err != nil {
		errorLog.Fatal(err)
	}
	app.eTag = etag

	var tlsConfig *tls.Config
	if cfg.tls.enabled {
		tlsConfig = &tls.Config{
			CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
			MinVersion:       tls.VersionTLS12,
		}
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		ErrorLog:     app.errorLog,
		Handler:      app.routes(),
		TLSConfig:    tlsConfig,
		IdleTimeout:  time.Minute,
		ReadTimeout:  900 * time.Second,
		WriteTimeout: 900 * time.Second,
	}

	app.infoLog.Printf("Starting server on %d", cfg.port)
	if cfg.tls.enabled {
		err = srv.ListenAndServeTLS(cfg.tls.certPath, cfg.tls.keyPath)
	} else {
		err = srv.ListenAndServe()
	}
	app.errorLog.Fatal(err)
}
