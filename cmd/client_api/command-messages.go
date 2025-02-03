package main

import (
	"bufio"
	"bytes"
	"email.mercata.com/internal/consts"
	"email.mercata.com/internal/email/address"
	linksPkg "email.mercata.com/internal/email/links"
	mcaPkg "email.mercata.com/internal/email/mca"
	messagePkg "email.mercata.com/internal/email/message"
	"email.mercata.com/internal/email/storage"
	userPkg "email.mercata.com/internal/email/user"
	noncePkg "email.mercata.com/internal/nonce"
	"email.mercata.com/internal/utils"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const ENDPOINT_REMOTE_MESSAGES_LIST = "/%s/%s/%s/link/%s/messages"
const ENDPOINT_REMOTE_MESSAGES_STREAM_LIST = "/%s/%s/%s/link/%s/streams/%s/messages"
const ENDPOINT_REMOTE_MESSAGES_FETCH = "/%s/%s/%s/link/%s/messages/%s"

const ENDPOINT_PRIVATE_MESSAGES_STATUS = "/%s/%s/%s/messages"
const ENDPOINT_PRIVATE_MESSAGES_STORE = "/%s/%s/%s/messages"
const ENDPOINT_PRIVATE_MESSAGES_DELETE = "/%s/%s/%s/messages/%s"

func messagesListCommand(args []string) {
	fs := flag.NewFlagSet("messages-list", flag.ExitOnError)
	accountEmail := fs.String("user", "", "use local profile of given user")
	authorEmail := fs.String("author", "", "fetch from remote user")
	streamID := fs.String("stream", "", "fetch for specific stream")
	hostOverride := fs.String("force-host", "", "enforce given host for the request")
	fs.Parse(args)

	var localUser *userPkg.User
	var err error

	if !address.ValidEmailAddress(*accountEmail) {
		fmt.Println("Error: not present or bad email address format")
		os.Exit(1)
	}

	localUser, err = userPkg.LocalUser(*accountEmail)
	if err != nil {
		fmt.Printf("Could not initialize local user '%s': %s\n", *accountEmail, err)
		os.Exit(1)
	}
	safeAuthorAddress, authorDomain, authorLocalPart := address.ParseEmailAddress(*authorEmail)

	link := linksPkg.Make(*accountEmail, safeAuthorAddress)

	var path string
	if *streamID != "" {
		path = fmt.Sprintf(ENDPOINT_REMOTE_MESSAGES_STREAM_LIST, consts.PUBLIC_API_PATH_PREFIX, authorDomain, authorLocalPart, link, *streamID)
	} else {
		path = fmt.Sprintf(ENDPOINT_REMOTE_MESSAGES_LIST, consts.PUBLIC_API_PATH_PREFIX, authorDomain, authorLocalPart, link)
	}

	var hosts []string
	if *hostOverride != "" {
		hosts = []string{*hostOverride}
	} else {
		hosts, err = mcaPkg.LookupEmailHosts(authorDomain, authorLocalPart)
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

		body, err := io.ReadAll(res.Body)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}
		fmt.Println(string(body))
	}
}

func messagesFetchCommand(args []string) {
	fs := flag.NewFlagSet("messages-fetch", flag.ExitOnError)
	accountEmail := fs.String("user", "", "use local profile of given user")
	authorEmail := fs.String("author", "", "fetch from remote user")
	messageID := fs.String("message-id", "", "message id to fetch")
	hostOverride := fs.String("force-host", "", "enforce given host for the request")
	fs.Parse(args)

	var localUser *userPkg.User
	var err error

	if !address.ValidEmailAddress(*accountEmail) {
		fmt.Println("Error: not present or bad email address format")
		os.Exit(1)
	}

	if !utils.ValidMessageID(*messageID) {
		fmt.Printf("Not a valid MessageID '%s'\n", *messageID)
		os.Exit(1)
	}

	localUser, err = userPkg.LocalUser(*accountEmail)
	if err != nil {
		fmt.Printf("Could not initialize local user '%s': %s\n", *accountEmail, err)
		os.Exit(1)
	}
	safeAuthorAddress, authorDomain, authorLocalPart := address.ParseEmailAddress(*authorEmail)

	link := linksPkg.Make(*accountEmail, safeAuthorAddress)
	path := fmt.Sprintf(ENDPOINT_REMOTE_MESSAGES_FETCH, consts.PUBLIC_API_PATH_PREFIX, authorDomain, authorLocalPart, link, *messageID)

	var hosts []string
	if *hostOverride != "" {
		hosts = []string{*hostOverride}
	} else {
		hosts, err = mcaPkg.LookupEmailHosts(authorDomain, authorLocalPart)
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

		messagePath, messageExists, err := storage.LocalTempMessageExists(safeAuthorAddress, *messageID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error verifying temp directory existence: %s\n", err)
			os.Exit(1)
		}
		if messageExists {
			fmt.Fprintf(os.Stderr, "Message already exist: %s\n", messagePath)
			os.Exit(1)
		}

		messagePath, err = storage.CreateLocalTempMessageDir(safeAuthorAddress, *messageID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
			os.Exit(1)
		}
		envelopePath, err := storage.LocalTempMessageEnvelopePath(safeAuthorAddress, *messageID)
		if err != nil {
			fmt.Println("Could not determine envelope path")
			os.Exit(1)
		}
		payloadPath, err := storage.LocalTempMessagePayloadPath(safeAuthorAddress, *messageID)
		if err != nil {
			fmt.Println("Could not determine payload path")
			os.Exit(1)
		}

		localPayloadFile, err := os.Create(payloadPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating file: %s\n", err)
			os.Exit(1)
		}
		defer localPayloadFile.Close()

		_, err = io.Copy(localPayloadFile, res.Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing payload body to file: %s\n", err)
			os.Exit(1)
		}

		envelope, err := messagePkg.MessageFromHeadersData(res.Header)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to parse response headers %s\n", err)
			os.Exit(1)
		}

		envelopeDumpStr := []byte(strings.Join(envelope.EnvelopeHeadersList, "\n"))
		err = ioutil.WriteFile(envelopePath, append(envelopeDumpStr, '\n'), 0644)
		if err != nil {
			fmt.Printf("failed to save request envelope: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("Message stored in %s on [%s]\n", messagePath, host)
	}
}

func messagesStatusCommand(args []string) {
	fs := flag.NewFlagSet("messages-status", flag.ExitOnError)
	accountEmail := fs.String("user", "", "fetch message status of given user")
	hostOverride := fs.String("force-host", "", "enforce given host for the request")
	fs.Parse(args)

	if !address.ValidEmailAddress(*accountEmail) {
		fmt.Println("Error: not present or bad email address format")
		os.Exit(1)
	}

	safeAddress, domain, localPart := address.ParseEmailAddress(*accountEmail)
	authorUser, err := userPkg.LocalUser(safeAddress)
	if err != nil {
		fmt.Printf("Could not initialize local user '%s': %s\n", safeAddress, err)
		os.Exit(1)
	}

	path := fmt.Sprintf(ENDPOINT_PRIVATE_MESSAGES_STATUS, consts.PRIVATE_API_PATH_PREFIX, domain, localPart)
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
		req, err := http.NewRequest("GET", uri.String(), nil)
		if err != nil {
			fmt.Printf("Local error: %s\n", err)
			os.Exit(1)
		}
		req.Header.Set("Content-Type", "text/plain")

		nonce, err := noncePkg.ForUser(authorUser)
		if err != nil {
			fmt.Printf("Local error: %s\n", err)
			os.Exit(1)
		}
		req.Header.Set(consts.AUTHORIZATION_HEADER_NONCE, noncePkg.ToHeader(nonce))

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
		body, err := io.ReadAll(res.Body)
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}
		fmt.Println(string(body))
	}
}

func messagesStoreCommand(args []string) {
	fs := flag.NewFlagSet("messages-store", flag.ExitOnError)
	accountEmail := fs.String("user", "", "update profile for given user")
	sourcePath := fs.String("message-path", "", "read message data from path")
	hostOverride := fs.String("force-host", "", "enforce given host for the request")
	fs.Parse(args)

	if !address.ValidEmailAddress(*accountEmail) {
		fmt.Println("Error: not present or bad email address format")
		os.Exit(1)
	}

	safeAddress, domain, localPart := address.ParseEmailAddress(*accountEmail)
	authorUser, err := userPkg.LocalUser(safeAddress)
	if err != nil {
		fmt.Printf("Could not initialize local user '%s': %s\n", safeAddress, err)
		os.Exit(1)
	}

	if *sourcePath == "" {
		fmt.Println("Error: must provide message source path")
		os.Exit(1)
	}
	messagePathExists, err := utils.FilePathExists(*sourcePath)
	if err != nil {
		fmt.Printf("Could not check message path '%s': %s\n", *sourcePath, err)
		os.Exit(1)
	}
	if !messagePathExists {
		fmt.Println("Error: message source path does not exist")
		os.Exit(1)
	}

	envelopePath := filepath.Join(*sourcePath, consts.MESSAGE_DIR_ENVELOPE_FILE_NAME)
	payloadPath := filepath.Join(*sourcePath, consts.MESSAGE_DIR_PAYLOAD_FILE_NAME)

	sourceEnvelopeData, err := ioutil.ReadFile(envelopePath)
	if err != nil {
		fmt.Printf("Error: could not read envelope %s\n", err)
		os.Exit(1)
	}
	message, err := messagePkg.ParseEnvelopeData(sourceEnvelopeData)
	if err != nil {
		fmt.Printf("Error: could not parse envelope %s\n", err)
		os.Exit(1)
	}

	path := fmt.Sprintf(ENDPOINT_PRIVATE_MESSAGES_STORE, consts.PRIVATE_API_PATH_PREFIX, domain, localPart)
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

	payloadFile, err := os.Open(payloadPath)
	if err != nil {
		fmt.Println("Error opening payload file:", err)
		os.Exit(1)
	}
	defer payloadFile.Close()

	for _, host := range hosts {
		client := http.Client{}
		pr, pw := io.Pipe()
		go func() {
			defer pw.Close()
			io.Copy(pw, payloadFile)
		}()
		if !strings.HasPrefix(host, "http") {
			host = "https://" + host
		}
		uri, err := url.ParseRequestURI(host + path)
		if err != nil {
			fmt.Print("Valid URL is required '%s': %s\n", uri, err)
			os.Exit(1)
		}
		fmt.Println("Trying: ", uri.String())
		req, err := http.NewRequest("POST", uri.String(), pr)
		if err != nil {
			fmt.Printf("Local error: %s\n", err)
			os.Exit(1)
		}

		nonce, err := noncePkg.ForUser(authorUser)
		if err != nil {
			fmt.Printf("Local error: %s\n", err)
			os.Exit(1)
		}

		if message.IsBroadcast && !message.IsFile() {
			req.Header.Set("Content-Type", "text/plain")
		} else {
			req.Header.Set("Content-Type", "application/octet-stream")
		}

		req.Header.Set(consts.AUTHORIZATION_HEADER_NONCE, noncePkg.ToHeader(nonce))
		err = writeEnvelopeAsRequestHeaders(sourceEnvelopeData, req)
		if err != nil {
			fmt.Printf("Local error: %s\n", err)
			os.Exit(1)
		}

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
		body, err := io.ReadAll(res.Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading response body: %s\n", err)
			os.Exit(1)
		}
		_, err = os.Stdout.Write(body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing response body to stdout: %s\n", err)
			os.Exit(1)
		}
	}
	fmt.Println("Message stored")
}

func writeEnvelopeAsRequestHeaders(envelopeContent []byte, req *http.Request) error {
	scanner := bufio.NewScanner(bytes.NewReader(envelopeContent))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line[0] == '#' {
			continue
		}
		parts := strings.SplitN(line, messagePkg.HEADER_KEY_VALUE_SEPARATOR, 2)
		if len(parts) != 2 {
			continue
		}
		if utils.ListContains(messagePkg.PERMITTED_ENVELOPE_KEYS, parts[0]) {
			req.Header.Set(parts[0], parts[1])
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func messagesDeleteCommand(args []string) {
	fs := flag.NewFlagSet("messages-delete", flag.ExitOnError)
	accountEmail := fs.String("user", "", "update profile for given user")
	messageID := fs.String("message-id", "", "message id to delete")
	hostOverride := fs.String("force-host", "", "enforce given host for the request")
	fs.Parse(args)

	if !address.ValidEmailAddress(*accountEmail) {
		fmt.Println("Error: not present or bad email address format")
		os.Exit(1)
	}

	safeAddress, domain, localPart := address.ParseEmailAddress(*accountEmail)
	authorUser, err := userPkg.LocalUser(safeAddress)
	if err != nil {
		fmt.Printf("Could not initialize local user '%s': %s\n", safeAddress, err)
		os.Exit(1)
	}

	if !utils.ValidMessageID(*messageID) {
		fmt.Printf("Not a valid MessageID '%s'\n", *messageID)
		os.Exit(1)
	}

	path := fmt.Sprintf(ENDPOINT_PRIVATE_MESSAGES_DELETE, consts.PRIVATE_API_PATH_PREFIX, domain, localPart, *messageID)
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

		nonce, err := noncePkg.ForUser(authorUser)
		if err != nil {
			fmt.Printf("Local error: %s\n", err)
			os.Exit(1)
		}

		req.Header.Set("Content-Type", "text/plain")
		req.Header.Set(consts.AUTHORIZATION_HEADER_NONCE, noncePkg.ToHeader(nonce))

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
			if res.StatusCode == http.StatusNotFound {
				fmt.Printf("Message [%s] not present on host [%s]\n", *messageID, host)
			} else {
				fmt.Printf("Response code: %d\n", res.StatusCode)
				os.Exit(1)
			}
		} else {
			fmt.Printf("Message [%s] deleted from host [%s]\n", *messageID, host)
		}
	}

}

func messagesAuthorCommand(args []string) {
	fs := flag.NewFlagSet("messages-author", flag.ExitOnError)
	authorEmailAddress := fs.String("author", "", "use the local user as author")
	readersEmailAddresses := fs.String("readers", "", "encrypt the message for the given readers")
	streamID := fs.String("stream-id", "", "assign the message to a stream of given id")
	subject := fs.String("subject", "", "set the message subject")
	subjectID := fs.String("subject-id", "", "reference an existing subject ID for the message, if any")
	category := fs.String("category", "personal", "define message category")
	parentMessageID := fs.String("part-of-id", "", "reference an existing message ID to which this is added") // Like in telegram a reply
	sourceOrBody := fs.String("body", "(empty body)", "use the content as a body or read from @FILEPATH")

	fs.Parse(args)

	if *authorEmailAddress == "" || !address.ValidEmailAddress(*authorEmailAddress) {
		fmt.Fprintf(os.Stderr, "Author email address is not valid: %s\n", *authorEmailAddress)
		os.Exit(1)
	}

	msg, err := messagePkg.NewMessage(*authorEmailAddress)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not make a message for '%s': %s\n", *authorEmailAddress, err)
		os.Exit(1)
	}

	if *subject != "" {
		if !messagePkg.ValidMessageSubject(*subject) {
			fmt.Fprintf(os.Stderr, "Provided 'Subject' is not a valid subject: %s\n", *subjectID)
			os.Exit(1)
		}
		msg.SetSubject(*subject)
	}

	if *subjectID != "" {
		if !utils.ValidMessageID(*subjectID) {
			fmt.Fprintf(os.Stderr, "Provided 'Subject-ID' is not a valid Message-ID: %s\n", *subjectID)
			os.Exit(1)
		}
		msg.SetSubjectID(*subjectID)
	}

	if msg.SubjectRequired() {
		fmt.Fprintf(os.Stderr, "'Subject' is required on new conversations\n")
		os.Exit(1)
	}

	if *category != "" {
		if !messagePkg.ValidCategory(*category) {
			fmt.Fprintf(os.Stderr, "Provided 'Category' is not valid: %s\n", *category)
			os.Exit(1)
		}
		msg.SetCategory(*category)
	}

	if *parentMessageID != "" {
		if !utils.ValidMessageID(*parentMessageID) {
			fmt.Fprintf(os.Stderr, "Provided 'Parent-Message-ID' is not a valid Message-ID: %s\n", *parentMessageID)
			os.Exit(1)
		}
		msg.SetParentMessageID(*parentMessageID)
	}

	if *streamID != "" {
		if !msg.SetStreamID(*streamID) {
			fmt.Fprintf(os.Stderr, "Provided 'StreamID' is not a acceptable name %s\n", *streamID)
			os.Exit(1)
		}
	}

	if *readersEmailAddresses != "" {
		readersAry := strings.Split(strings.TrimSpace(*readersEmailAddresses), ",")
		for _, readerAddress := range readersAry {
			if readerAddress != "" {
				if !address.ValidEmailAddress(readerAddress) {
					fmt.Fprintf(os.Stderr, "Invalid reader '%s'\n", readerAddress)
					os.Exit(1)
				}
				err := msg.AddReader(readerAddress)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Could not add reader '%s': %s\n", readerAddress, err)
					os.Exit(1)
				}
			}
		}
	}

	trimmedSourceOrBody := strings.TrimSpace(*sourceOrBody)
	if trimmedSourceOrBody[0] == '@' {
		absoluteSourcePath, err := filepath.Abs(trimmedSourceOrBody[1:])
		if err != nil {
			fmt.Printf("Bad path '%s': %s\n", absoluteSourcePath, err)
			os.Exit(1)
		}
		err = msg.SetFileContent(absoluteSourcePath)
		if err != nil {
			fmt.Printf("Error accessing file '%s': %s\n", absoluteSourcePath, err)
			os.Exit(1)
		}
	} else {
		contentBody := []byte(trimmedSourceOrBody)
		// TODO: Is it reall plain body? or Markdown? or Bytes?
		msg.SetPlainContent(contentBody)
	}

	destFilePath, err := msg.Seal()
	if err != nil {
		fmt.Printf("Error writing file '%s': %s\n", destFilePath, err)
		os.Exit(1)
	}
	fmt.Println("\nEnvelope ready at:\n==================\n", destFilePath, "\n\n")
}

func messagesOpenCommand(args []string) {
	fs := flag.NewFlagSet("messages-open", flag.ExitOnError)
	readerEmail := fs.String("local-reader", "", "use the given account as local message reader")
	authorEmail := fs.String("remote-author", "", "use the given account as remote message author")
	messageDirPath := fs.String("message-dir", "", "read message at the given path")
	fs.Parse(args)

	if *readerEmail == "" || !address.ValidEmailAddress(*readerEmail) {
		fmt.Println("Valid reader email address is required.")
		os.Exit(1)
	}

	if *authorEmail == "" || !address.ValidEmailAddress(*authorEmail) {
		fmt.Println("Valid author email address is required.")
		os.Exit(1)
	}

	absoluteMessagePath, err := filepath.Abs(*messageDirPath)
	if err != nil {
		fmt.Printf("Bad path '%s': %s\n", *messageDirPath, err)
		os.Exit(1)
	}

	authorUser, err := userPkg.LocalUser(*authorEmail)
	if err != nil {
		fmt.Printf("Could not initialize local author '%s': %s\n", *authorEmail, err)
		os.Exit(1)
	}

	readerUser, err := userPkg.LocalUser(*readerEmail)
	if err != nil {
		fmt.Printf("Could not initialize local reader '%s': %s\n", *readerEmail, err)
		os.Exit(1)
	}
	reader := userPkg.AsReader(readerUser)

	_, err = messagePkg.Open(absoluteMessagePath, authorUser, reader)
	if err != nil {
		fmt.Printf("Message could not be loaded from path '%s': %s\n", absoluteMessagePath, err)
		os.Exit(1)
	}
	fmt.Println("\nMessage opened at:\n==================\n", absoluteMessagePath, "\n\n")
}
