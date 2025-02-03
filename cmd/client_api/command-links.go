package main

import (
	"bufio"
	"email.mercata.com/internal/consts"
	"email.mercata.com/internal/crypto"
	"email.mercata.com/internal/email/address"
	linksPkg "email.mercata.com/internal/email/links"
	mcaPkg "email.mercata.com/internal/email/mca"
	userPkg "email.mercata.com/internal/email/user"
	noncePkg "email.mercata.com/internal/nonce"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const ENDPOINT_LOCAL_LINKS_STORE = "/%s/%s/%s/links/%s"
const ENDPOINT_LOCAL_LINKS_DELETE = "/%s/%s/%s/links/%s"
const ENDPOINT_LOCAL_LINKS_LIST = "/%s/%s/%s/links"

// go run cmd/client/* links-make -user me@dejanstrbac.com -contact dejan@mercata.com
func linksMakeCommand(args []string) {
	fs := flag.NewFlagSet("links-make", flag.ExitOnError)
	accountEmail := fs.String("user", "", "email address of the local user")
	readerEmail := fs.String("contact", "", "contact's email address")
	fs.Parse(args)

	if !address.ValidEmailAddress(*accountEmail) {
		fmt.Println("Error: not present or bad user email address format")
		os.Exit(1)
	}

	if !address.ValidEmailAddress(*readerEmail) {
		fmt.Println("Error: not present or bad contact email address format")
		os.Exit(1)
	}

	link := linksPkg.Make(*accountEmail, *readerEmail)
	fmt.Printf("Link(%s, %s): %s\n", *accountEmail, *readerEmail, link)
}

// go run cmd/client/* links-list -user me@dejanstrbac.com -force-host http://127.0.0.1:4000
func linksListCommand(args []string) {
	fs := flag.NewFlagSet("links-list", flag.ExitOnError)
	accountEmail := fs.String("user", "", "use local profile of given user")
	hostOverride := fs.String("force-host", "", "enforce given host for the request")
	fs.Parse(args)

	var localUser *userPkg.User
	var err error

	if !address.ValidEmailAddress(*accountEmail) {
		fmt.Println("Error: not present or bad email address format")
		os.Exit(1)
	}

	safeUserAddress, userDomain, userLocalPart := address.ParseEmailAddress(*accountEmail)
	localUser, err = userPkg.LocalUser(safeUserAddress)
	if err != nil {
		fmt.Printf("Could not initialize local user '%s': %s\n", safeUserAddress, err)
		os.Exit(1)
	}

	path := fmt.Sprintf(ENDPOINT_LOCAL_LINKS_LIST, consts.PRIVATE_API_PATH_PREFIX, userDomain, userLocalPart)

	var hosts []string
	if *hostOverride != "" {
		hosts = []string{*hostOverride}
	} else {
		hosts, err = mcaPkg.LookupEmailHosts(userDomain, userLocalPart)
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
		req, err := http.NewRequest("GET", uri.String(), nil)
		if err != nil {
			fmt.Printf("Local error: %s\n", err)
			os.Exit(1)
		}
		req.Header.Set("Content-Type", "text/plain")
		n, err := noncePkg.ForUser(localUser)
		if err != nil {
			fmt.Printf("Local error: %s\n", err)
			os.Exit(1)
		}
		req.Header.Set(consts.AUTHORIZATION_HEADER_NONCE, noncePkg.ToHeader(n))

		res, err := client.Do(req)
		if err != nil {
			if netErr, ok := err.(*net.OpError); ok && netErr.Timeout() {
				fmt.Fprintf(os.Stderr, "Timeout error: %s\n", err)
			} else if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
				fmt.Fprintf(os.Stderr, "URL timeout error: %s\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "Other error: %s\n", err)
			}
			fmt.Fprintf(os.Stderr, "Could not query URL: %s\n", err)
			os.Exit(1)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			fmt.Printf("Response code: %d\n", res.StatusCode)
			os.Exit(1)
		}

		scanner := bufio.NewScanner(res.Body)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			contact, err := crypto.DecryptAnonymous(localUser.PrivateEncryptionKey, localUser.PublicEncryptionKey, line)
			if err != nil {
				fmt.Println("line could not be decrypted: ", line)
				continue
			}
			fmt.Println(string(contact))
		}
	}
}

// go run cmd/client/* links-store -user me@dejanstrbac.com -contact dejan@mercata.com -force-host http://127.0.0.1:4000
func linksStoreCommand(args []string) {
	fs := flag.NewFlagSet("links-store", flag.ExitOnError)
	accountEmail := fs.String("user", "", "email addres of local user")
	contactEmail := fs.String("contact", "", "contact email address to save")
	hostOverride := fs.String("force-host", "", "enforce given host for the request")
	fs.Parse(args)

	if *accountEmail == "" || !address.ValidEmailAddress(*accountEmail) {
		fmt.Println("Local user is not a valid email address.")
		os.Exit(1)
	}

	if *contactEmail == "" || !address.ValidEmailAddress(*contactEmail) {
		fmt.Println("Contact is not a valid email address.")
		os.Exit(1)
	}

	localUser, err := userPkg.LocalUser(*accountEmail)
	if err != nil {
		fmt.Printf("Could not initialize local user '%s': %s\n", *accountEmail, err)
		os.Exit(1)
	}

	safeContactAddress, _, _ := address.ParseEmailAddress(*contactEmail)
	contactAddressEncrypted, err := crypto.EncryptAnonymous(localUser.PublicEncryptionKey, []byte(safeContactAddress))
	if err != nil {
		fmt.Printf("Could not encrypt contact address '%s': %s\n", localUser.Address, err)
		os.Exit(1)
	}

	safeAccountEmail, domain, localPart := address.ParseEmailAddress(*accountEmail)
	link := linksPkg.Make(safeAccountEmail, safeContactAddress)
	path := fmt.Sprintf(ENDPOINT_LOCAL_LINKS_STORE, consts.PRIVATE_API_PATH_PREFIX, domain, localPart, link)

	var hosts []string
	if *hostOverride != "" {
		hosts = []string{*hostOverride}
	} else {
		hosts, err = mcaPkg.LookupEmailHosts(domain, localPart)
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
		req, err := http.NewRequest("PUT", uri.String(), strings.NewReader(contactAddressEncrypted))
		if err != nil {
			fmt.Printf("Local error: %s\n", err)
			os.Exit(1)
		}
		req.Header.Set("Content-Type", "text/plain")
		n, err := noncePkg.ForUser(localUser)
		if err != nil {
			fmt.Printf("Local error: %s\n", err)
			os.Exit(1)
		}
		req.Header.Set(consts.AUTHORIZATION_HEADER_NONCE, noncePkg.ToHeader(n))

		res, err := client.Do(req)
		if err != nil {
			if netErr, ok := err.(*net.OpError); ok && netErr.Timeout() {
				fmt.Fprintf(os.Stderr, "Timeout error: %s\n", err)
			} else if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
				fmt.Fprintf(os.Stderr, "URL timeout error: %s\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "Other error: %s\n", err)
			}
			fmt.Fprintf(os.Stderr, "Could not query URL: %s\n", err)
			os.Exit(1)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "Response code: %d\n", res.StatusCode)
			os.Exit(1)
		}
		fmt.Printf("Stored contact %s at %s", safeContactAddress, host)
	}
}

// go run cmd/client/* links-delete -user me@dejanstrbac.com -contact dejan@mercata.com -force-host http://127.0.0.1:4000
func linksDeleteCommand(args []string) {
	fs := flag.NewFlagSet("links-store", flag.ExitOnError)
	accountEmail := fs.String("user", "", "email addres of local user")
	contactEmail := fs.String("contact", "", "contact email address to save")
	hostOverride := fs.String("force-host", "", "enforce given host for the request")
	fs.Parse(args)

	if *accountEmail == "" || !address.ValidEmailAddress(*accountEmail) {
		fmt.Println("Local user is not a valid email address.")
		os.Exit(1)
	}

	if *contactEmail == "" || !address.ValidEmailAddress(*contactEmail) {
		fmt.Println("Contact is not a valid email address.")
		os.Exit(1)
	}

	localUser, err := userPkg.LocalUser(*accountEmail)
	if err != nil {
		fmt.Printf("Could not initialize local user '%s': %s\n", *accountEmail, err)
		os.Exit(1)
	}

	safeContactAddress, _, _ := address.ParseEmailAddress(*contactEmail)
	safeAccountEmail, domain, localPart := address.ParseEmailAddress(*accountEmail)
	link := linksPkg.Make(safeAccountEmail, safeContactAddress)
	path := fmt.Sprintf(ENDPOINT_LOCAL_LINKS_DELETE, consts.PRIVATE_API_PATH_PREFIX, domain, localPart, link)

	var hosts []string
	if *hostOverride != "" {
		hosts = []string{*hostOverride}
	} else {
		hosts, err = mcaPkg.LookupEmailHosts(domain, localPart)
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
		req, err := http.NewRequest("DELETE", uri.String(), nil)
		if err != nil {
			fmt.Printf("Local error: %s\n", err)
			os.Exit(1)
		}
		req.Header.Set("Content-Type", "text/plain")
		n, err := noncePkg.ForUser(localUser)
		if err != nil {
			fmt.Printf("Local error: %s\n", err)
			os.Exit(1)
		}
		req.Header.Set(consts.AUTHORIZATION_HEADER_NONCE, noncePkg.ToHeader(n))

		res, err := client.Do(req)
		if err != nil {
			if netErr, ok := err.(*net.OpError); ok && netErr.Timeout() {
				fmt.Fprintf(os.Stderr, "Timeout error: %s\n", err)
			} else if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
				fmt.Fprintf(os.Stderr, "URL timeout error: %s\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "Other error: %s\n", err)
			}
			fmt.Fprintf(os.Stderr, "Could not query URL: %s\n", err)
			os.Exit(1)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "Response code: %d\n", res.StatusCode)
			os.Exit(1)
		}
		fmt.Printf("Deleted contact %s from %s", safeContactAddress, host)
	}
}
