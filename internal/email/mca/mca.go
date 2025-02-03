package mca

import (
	"email.mercata.com/internal/consts"
	"email.mercata.com/internal/utils"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const WELL_KNOWN_URI = ".well-known/mail.txt"

func LookupEmailHosts(domainName, localPart string) ([]string, error) {
	var mailHosts []string

	var body []byte
	var err error

	defaultMailHostname := fmt.Sprintf("mail.%s", domainName)
	body, err = tryWellKnownHost(domainName)
	wellKnownBodyPresent := true
	if err != nil {
		body, err = tryWellKnownHost(defaultMailHostname)
		if err != nil {
			wellKnownBodyPresent = false
		}
	}

	var validHosts []string

	if wellKnownBodyPresent {
		lines := strings.Split(string(body), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if len(line) == 0 || line[0] == '#' {
				continue
			}
			validHosts = append(validHosts, line)
		}

		validHosts = utils.FilterValidHostnames(validHosts)

		if len(validHosts) == 0 {
			validHosts = append(validHosts, defaultMailHostname)
		}
	}

	// Filter only hostnames that respond to the given address
	for _, host := range validHosts {
		profileFound, err := tryDomainDelegation(host, domainName)
		if err != nil {
			continue
		}
		if profileFound {
			mailHosts = append(mailHosts, host)
		}
	}

	return mailHosts, nil
}

func tryWellKnownHost(hostname string) ([]byte, error) {
	wellKnownURI := fmt.Sprintf("https://%s/%s", hostname, WELL_KNOWN_URI)
	resp, err := http.Get(wellKnownURI)
	defer resp.Body.Close()
	if err != nil {
		if netErr, ok := err.(*net.OpError); ok && netErr.Timeout() {
			fmt.Println("Timeout error:", err)
		} else if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
			fmt.Println("URL timeout error:", err)
		} else {
			fmt.Println("Other error:", err)
		}
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}
	return body, nil
}

func tryDomainDelegation(hostname, domainName string) (bool, error) {
	delegationURI := fmt.Sprintf("https://%s/%s/%s", hostname, consts.PUBLIC_API_PATH_PREFIX, domainName)
	resp, err := http.Head(delegationURI)
	if err != nil {
		if netErr, ok := err.(*net.OpError); ok && netErr.Timeout() {
			fmt.Println("Timeout error:", err)
		} else if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
			fmt.Println("URL timeout error:", err)
		} else {
			fmt.Println("Other error:", err)
		}
		return false, err
	}

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	return true, nil
}
