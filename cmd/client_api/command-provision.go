package main

import (
	"bytes"
	"email.mercata.com/internal/consts"
	"email.mercata.com/internal/email/address"
	"email.mercata.com/internal/email/mca"
	"email.mercata.com/internal/email/user"
	"email.mercata.com/internal/nonce"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func provisionCommand(args []string) {
	fs := flag.NewFlagSet("provision", flag.ExitOnError)
	accountEmail := fs.String("user", "", "provision new account with address")
	sourceFile := fs.String("source-file", "", "read profile from given file")
	hostOverride := fs.String("force-host", "", "enforce given host for the request")
	fs.Parse(args)

	if !address.ValidEmailAddress(*accountEmail) {
		fmt.Println("Error: not present or bad email address format")
		os.Exit(1)
	}

	safeAddress, domain, localPart := address.ParseEmailAddress(*accountEmail)
	authorUser, err := user.LocalUser(safeAddress)
	if err != nil {
		fmt.Printf("Could not initialize local user '%s': %s\n", safeAddress, err)
		os.Exit(1)
	}

	if *sourceFile == "" {
		fmt.Println("Error: must provide source file")
		os.Exit(1)
	}
	sourceProfileData, err := ioutil.ReadFile(*sourceFile)
	if err != nil {
		fmt.Println("Error: could not read source file")
		os.Exit(1)
	}

	path := fmt.Sprintf("/%s/%s/%s", consts.PRIVATE_PROVISION_PATH_PREFIX, domain, localPart)
	var hosts []string
	if *hostOverride != "" {
		hosts = []string{*hostOverride}
	} else {
		hosts, err = mca.LookupEmailHosts(domain, localPart)
		if err != nil || len(hosts) == 0 {
			fmt.Println("No hosts to contact")
			os.Exit(1)
		}
	}

	for _, host := range hosts {
		client := http.Client{}
		if !strings.HasPrefix(host, "http") {
			host = "https://" + host
		}
		uri, err := url.ParseRequestURI(host + path)
		if err != nil {
			fmt.Print("Valid URL is required '%s': %s\n", uri, err)
			os.Exit(1)
		}
		fmt.Println("Trying: ", uri.String())
		req, err := http.NewRequest("POST", uri.String(), bytes.NewReader(sourceProfileData))
		if err != nil {
			fmt.Printf("Local error: %s\n", err)
			os.Exit(1)
		}
		req.Header.Set("Content-Type", "text/plain")

		n, err := nonce.ForUser(authorUser)
		if err != nil {
			fmt.Printf("Local error: %s\n", err)
			os.Exit(1)
		}

		req.Header = http.Header{
			consts.AUTHORIZATION_HEADER_NONCE: {nonce.ToHeader(n)},
		}
		res, err := client.Do(req)
		fmt.Println("Response:", res.Status)
	}
}
