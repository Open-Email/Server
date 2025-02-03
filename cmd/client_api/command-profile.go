package main

import (
	"bytes"
	"email.mercata.com/internal/consts"
	"email.mercata.com/internal/email/address"
	"email.mercata.com/internal/email/mca"
	profilePkg "email.mercata.com/internal/email/profile"
	"email.mercata.com/internal/email/user"
	"email.mercata.com/internal/nonce"
	"email.mercata.com/internal/utils"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func profileFetchCommand(args []string) {
	fs := flag.NewFlagSet("profile-fetch", flag.ExitOnError)
	accountEmail := fs.String("user", "", "lookup keys for given user")
	fs.Parse(args)

	if !address.ValidEmailAddress(*accountEmail) {
		fmt.Println("Error: not present or bad email address format")
		os.Exit(1)
	}

	safeAddress, domain, user := address.ParseEmailAddress(*accountEmail)

	profile, err := profilePkg.GetRemoteProfile(safeAddress, domain, user)
	if err != nil {
		fmt.Printf("Profile could not be read: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("Address:       ", profile.User.Address)
	fmt.Println("Name:          ", "\""+profile.Name+"\"")
	fmt.Println("EncryptionKey: ", profile.User.PublicEncryptionKeyBase64)
	fmt.Println("SigningKey:    ", profile.User.PublicSigningKeyBase64)
	fmt.Println("----------")

	fmt.Println(string(*profile.RemoteBody))
}

func profileStoreCommand(args []string) {
	fs := flag.NewFlagSet("profile-store", flag.ExitOnError)
	accountEmail := fs.String("user", "", "update profile for given user")
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

	path := fmt.Sprintf("/%s/%s/%s/profile", consts.PRIVATE_API_PATH_PREFIX, domain, localPart)
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
		req, err := http.NewRequest("PUT", uri.String(), bytes.NewReader(sourceProfileData))
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

func profileImageFetchCommand(args []string) {
	fs := flag.NewFlagSet("profile-image-fetch", flag.ExitOnError)
	accountEmail := fs.String("user", "", "lookup image for given user")
	outputFilePath := fs.String("output-dir", "", "save image in directory")
	fs.Parse(args)

	if !address.ValidEmailAddress(*accountEmail) {
		fmt.Println("Error: not present or bad email address format")
		os.Exit(1)
	}

	if *outputFilePath == "" {
		fmt.Println("'-output' file path is required")
		os.Exit(1)
	}

	safeAddress, domain, user := address.ParseEmailAddress(*accountEmail)

	profileImageData, err := profilePkg.GetRemoteProfileImage(domain, user)
	if err != nil {
		fmt.Printf("Profile image could not be read: %s\n", err)
		os.Exit(1)
	}

	mimeType, err := utils.DetermineFileTypeOfData(profileImageData)
	if err != nil {
		fmt.Printf("Profile image mime type could not be determined: %s\n", err)
		os.Exit(1)
	}
	extension := strings.SplitN(mimeType, "/", 2)
	filename := safeAddress + "." + extension[1]
	imagePath, err := filepath.Abs(filepath.Join(*outputFilePath, filename))
	if err != nil {
		fmt.Printf("Bad output-dir path: %s\n", err)
		os.Exit(1)
	}
	err = ioutil.WriteFile(imagePath, *profileImageData, 0644)
	if err != nil {
		fmt.Printf("Could not save profile image: %s\n", err)
		os.Exit(1)
	}

	fmt.Println("Profile image saved at:", imagePath)
}

func profileImageStoreCommand(args []string) {
	fs := flag.NewFlagSet("profile-image-store", flag.ExitOnError)
	accountEmail := fs.String("user", "", "update profile for given user")
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
		fmt.Println("Error: must provide image source file")
		os.Exit(1)
	}

	sourceProfileImageData, err := ioutil.ReadFile(*sourceFile)
	if err != nil {
		fmt.Println("Error: could not read source file")
		os.Exit(1)
	}
	mimeType, err := utils.DetermineFileTypeOfData(&sourceProfileImageData)
	if err != nil {
		fmt.Printf("Source image mime type could not be determined: %s\n", err)
		os.Exit(1)
	}
	if !profilePkg.ImageMimeTypeIsPermitted(mimeType) {
		fmt.Printf("Source image mime type is not permitted: %s\n", mimeType)
		os.Exit(1)
	}

	path := fmt.Sprintf("/%s/%s/%s/image", consts.PRIVATE_API_PATH_PREFIX, domain, localPart)
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
		req, err := http.NewRequest("PUT", uri.String(), bytes.NewReader(sourceProfileImageData))
		if err != nil {
			fmt.Printf("Local error: %s\n", err)
			os.Exit(1)
		}
		req.Header.Set("Content-Type", mimeType)

		n, err := nonce.ForUser(authorUser)
		if err != nil {
			fmt.Printf("Local error: %s\n", err)
			os.Exit(1)
		}

		req.Header.Set(consts.AUTHORIZATION_HEADER_NONCE, nonce.ToHeader(n))

		res, err := client.Do(req)
		fmt.Println("Response:", res.Status)
	}
}
