package main

import (
	"email.mercata.com/internal/crypto"
	"email.mercata.com/internal/email/address"
	"email.mercata.com/internal/email/keys"
	"email.mercata.com/internal/email/profile"
	"flag"
	"fmt"
	"os"
)

func keysGenCommand(args []string) {
	fs := flag.NewFlagSet("keys-gen", flag.ExitOnError)

	generateEncKeyPair := fs.Bool("encryption", true, "generates only a new encryption key pair")
	generateSigKeyPair := fs.Bool("signing", true, "generates only a new signing key pair")
	generateForceOverwrite := fs.Bool("overwrite", false, "overwriting existing keys")
	accountEmail := fs.String("account", "", "saves generated key pair for given account")

	fs.Parse(args)

	if *accountEmail == "" {
		fmt.Println("Account email address is required.")
		os.Exit(1)
	}

	if !address.ValidEmailAddress(*accountEmail) {
		fmt.Println("Bad email address format given for account")
		os.Exit(1)
	}

	safeAddress, _, _ := address.ParseEmailAddress(*accountEmail)

	fmt.Println()

	if *generateEncKeyPair {
		privKeyEncodedStr, pubKeyEncodedStr := crypto.GenerateEncryptionKeys()
		privKeyPath, err := keys.StoreLocalEncryptionPrivateKey(safeAddress, privKeyEncodedStr, *generateForceOverwrite)
		if err != nil {
			fmt.Println("Could not save private encryption key: ", err)
			os.Exit(1)
		}
		pubKeyPath, err := keys.StoreLocalEncryptionPublicKey(safeAddress, pubKeyEncodedStr, *generateForceOverwrite)
		if err != nil {
			fmt.Println("Could not save public encryption key: ", err)
			os.Exit(1)
		}

		fmt.Printf(" Encryption Keys (%s)\n", safeAddress)
		fmt.Println(" ==============================================================")
		fmt.Println(" Public \t" + pubKeyEncodedStr)
		fmt.Println()
		fmt.Println("        \t" + privKeyPath)
		fmt.Println("        \t" + pubKeyPath + "\n")
	}

	if *generateSigKeyPair {
		privKeyEncodedStr, pubKeyEncodedStr := crypto.GenerateSigningKeys()
		privKeyPath, err := keys.StoreLocalSigningPrivateKey(safeAddress, privKeyEncodedStr, *generateForceOverwrite)
		if err != nil {
			fmt.Println("Could not save private signing key: ", err)
			os.Exit(1)
		}
		pubKeyPath, err := keys.StoreLocalSigningPublicKey(safeAddress, pubKeyEncodedStr, *generateForceOverwrite)
		if err != nil {
			fmt.Println("Could not save public signing key: ", err)
			os.Exit(1)
		}
		fmt.Printf(" Signing Keys (%s)\n", safeAddress)
		fmt.Println(" ==============================================================")
		fmt.Println(" Public \t" + pubKeyEncodedStr)
		fmt.Println()
		fmt.Println("        \t" + privKeyPath)
		fmt.Println("        \t" + pubKeyPath + "\n")
	}
}

func keysLookupCommand(args []string) {
	fs := flag.NewFlagSet("keys-lookup", flag.ExitOnError)

	accountEmail := fs.String("account", "", "looksup keys for given account")
	fs.Parse(args)

	// The account email is required for any operation
	if !address.ValidEmailAddress(*accountEmail) {
		fmt.Println("Error: not present or bad email address format")
		os.Exit(1)
	}

	safeAddress, domain, user := address.ParseEmailAddress(*accountEmail)

	profile, err := profile.GetRemoteProfile(safeAddress, domain, user)
	if err != nil || profile == nil {
		fmt.Println("Error: profile could not be read")
		return
	}

	if profile.Address != safeAddress {
		fmt.Println("Error: profile does not match requested address")
		os.Exit(1)
	}
	fmt.Println("Address:       ", profile.User.Address)
	fmt.Println("Name:          ", "\""+profile.Name+"\"")
	fmt.Println("EncryptionKey: ", profile.User.PublicEncryptionKeyBase64)
	fmt.Println("SigningKey:    ", profile.User.PublicSigningKeyBase64)
	fmt.Println("SigningKeyFingerprint:    ", profile.User.PublicSigningKeyFingerprint)
}
