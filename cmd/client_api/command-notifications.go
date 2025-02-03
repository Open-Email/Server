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
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const ENDPOINT_REMOTE_NOTIFICATION_STORE = "/%s/%s/%s/link/%s/notifications"
const ENDPOINT_LOCAL_NOTIFICATION_LIST = "/%s/%s/%s/notifications"

func notificationsListCommand(args []string) {
	fs := flag.NewFlagSet("notifications-list", flag.ExitOnError)
	accountEmail := fs.String("user", "", "use local profile of given user")
	processNotifications := fs.Bool("process", false, "process notifications after fetching them")
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

	path := fmt.Sprintf(ENDPOINT_LOCAL_NOTIFICATION_LIST, consts.PRIVATE_API_PATH_PREFIX, userDomain, userLocalPart)

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

		if !*processNotifications {
			body, err := io.ReadAll(res.Body)
			if err != nil {
				fmt.Printf("Error: %s\n", err)
				os.Exit(1)
			}
			fmt.Println(string(body))
			return
		}

		scanner := bufio.NewScanner(res.Body)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			notificationParts := strings.SplitN(line, ",", 2)

			decryptedData, err := crypto.DecryptAnonymous(localUser.PrivateEncryptionKey, localUser.PublicEncryptionKey, notificationParts[1])

			if err != nil {
				fmt.Printf("Error: %s\n", err)
				os.Exit(1)
			}

			authorAddress := string(decryptedData)
			link := linksPkg.Make(safeUserAddress, authorAddress)
			if link != notificationParts[0] {
				fmt.Printf("Link mismatch on notification from %s\n", authorAddress)
				continue
			}

			fmt.Printf("Verified: %s\n", authorAddress)
		}

	}
}

func notificationsStoreCommand(args []string) {
	fs := flag.NewFlagSet("notifications-store", flag.ExitOnError)
	accountEmail := fs.String("user", "", "use local profile of given user")
	readerEmail := fs.String("reader", "", "notify remote user")
	hostOverride := fs.String("force-host", "", "enforce given host for the request")
	fs.Parse(args)

	if *accountEmail == "" || !address.ValidEmailAddress(*accountEmail) {
		fmt.Println("Local user not valid.")
		os.Exit(1)
	}

	if *readerEmail == "" || !address.ValidEmailAddress(*readerEmail) {
		fmt.Println("Reader not valid.")
		os.Exit(1)
	}

	localUser, err := userPkg.LocalUser(*accountEmail)
	if err != nil {
		fmt.Printf("Could not initialize local user '%s': %s\n", *accountEmail, err)
		os.Exit(1)
	}

	readerUser, err := userPkg.LocalUser(*readerEmail) // TODO: fetch from remote
	if err != nil {
		fmt.Printf("Could not initialize remote user '%s': %s\n", *readerEmail, err)
		os.Exit(1)
	}

	// Notifications include own address of the notifier, encrypted
	callerAddress := []byte(localUser.Address)
	callerAddressEncrypted, err := crypto.EncryptAnonymous(readerUser.PublicEncryptionKey, callerAddress)
	if err != nil {
		fmt.Printf("Could not encrypt caller address '%s': %s\n", localUser.Address, err)
		os.Exit(1)
	}

	safeReaderAddress, readerDomain, readerLocalPart := address.ParseEmailAddress(*readerEmail)
	link := linksPkg.Make(*accountEmail, safeReaderAddress)
	path := fmt.Sprintf(ENDPOINT_REMOTE_NOTIFICATION_STORE, consts.PUBLIC_API_PATH_PREFIX, readerDomain, readerLocalPart, link)

	var hosts []string
	if *hostOverride != "" {
		hosts = []string{*hostOverride}
	} else {
		hosts, err = mcaPkg.LookupEmailHosts(readerDomain, readerLocalPart)
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
		req, err := http.NewRequest("PUT", uri.String(), strings.NewReader(callerAddressEncrypted))
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
			if res.StatusCode == http.StatusForbidden {
				fmt.Printf("Remote reader [%s] does not accept non-contact notifications\n", *readerEmail)
				os.Exit(0)
			}
			fmt.Fprintf(os.Stderr, "Response code: %d\n", res.StatusCode)
			os.Exit(1)
		}
		fmt.Printf("Notified user %s at %s", *readerEmail, host)
	}
}
