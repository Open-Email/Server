package keys

import (
	"email.mercata.com/internal/crypto"
	"email.mercata.com/internal/email/address"
	"email.mercata.com/internal/email/storage"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const PUBLIC_ENCRYPTION_KEY_SUFFIX = "encrypt.public"
const PRIVATE_ENCRYPTION_KEY_SUFFIX = "encrypt.private"
const PRIVATE_SIGNING_KEY_SUFFIX = "sign.private"
const PUBLIC_SIGNING_KEY_SUFFIX = "sign.public"
const PREVIOUS_KEY_SUFFIX = ".previous"

func StoreLocalEncryptionPublicKey(emailAddress, data string, overwrite bool) (string, error) {
	return StoreLocalKey(emailAddress, PUBLIC_ENCRYPTION_KEY_SUFFIX, data, overwrite)
}
func StoreLocalEncryptionPrivateKey(emailAddress, data string, overwrite bool) (string, error) {
	return StoreLocalKey(emailAddress, PRIVATE_ENCRYPTION_KEY_SUFFIX, data, overwrite)
}
func StoreLocalSigningPrivateKey(emailAddress, data string, overwrite bool) (string, error) {
	return StoreLocalKey(emailAddress, PRIVATE_SIGNING_KEY_SUFFIX, data, overwrite)
}
func StoreLocalSigningPublicKey(emailAddress, data string, overwrite bool) (string, error) {
	return StoreLocalKey(emailAddress, PUBLIC_SIGNING_KEY_SUFFIX, data, overwrite)
}

func StoreLocalKey(emailAddress, suffix, data string, overwrite bool) (string, error) {
	err := storage.CreateEmailHomePath(emailAddress)
	if err != nil {
		return "", err
	}

	keyFilename := emailAddress + "." + suffix
	homePath, err := storage.LocalHomePath(emailAddress)
	if err != nil {
		return "", err
	}
	keyPath := filepath.Join(homePath, keyFilename)
	if !overwrite {
		if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
			return keyPath, errors.New("Fatal error: key exists already")
		}
	}
	return keyPath, os.WriteFile(keyPath, []byte(data), 0644)
}

func GetLocalEncryptionPublicKey(emailAddress string) (string, [32]byte, error) {
	return GetLocalKey32(emailAddress, PUBLIC_ENCRYPTION_KEY_SUFFIX)
}
func GetLocalEncryptionPrivateKey(emailAddress string) (string, [32]byte, error) {
	return GetLocalKey32(emailAddress, PRIVATE_ENCRYPTION_KEY_SUFFIX)
}
func GetLocalSigningPublicKey(emailAddress string) (string, [32]byte, error) {
	return GetLocalKey32(emailAddress, PUBLIC_SIGNING_KEY_SUFFIX)
}
func GetLocalSigningPrivateKey(emailAddress string) (string, [64]byte, error) {
	return GetLocalKey64(emailAddress, PRIVATE_SIGNING_KEY_SUFFIX)
}
func GetLocalPreviousEncryptionPublicKey(emailAddress string) (string, [32]byte, error) {
	return GetLocalKey32(emailAddress, PUBLIC_ENCRYPTION_KEY_SUFFIX+PREVIOUS_KEY_SUFFIX)
}
func GetLocalPreviousEncryptionPrivateKey(emailAddress string) (string, [32]byte, error) {
	return GetLocalKey32(emailAddress, PRIVATE_ENCRYPTION_KEY_SUFFIX+PREVIOUS_KEY_SUFFIX)
}
func GetLocalPreviousSigningPublicKey(emailAddress string) (string, [32]byte, error) {
	return GetLocalKey32(emailAddress, PUBLIC_SIGNING_KEY_SUFFIX+PREVIOUS_KEY_SUFFIX)
}
func GetLocalPreviousSigningPrivateKey(emailAddress string) (string, [64]byte, error) {
	return GetLocalKey64(emailAddress, PRIVATE_SIGNING_KEY_SUFFIX+PREVIOUS_KEY_SUFFIX)
}

func GetLocalKey32(emailAddress, suffix string) (string, [32]byte, error) {
	var keyBytes [32]byte

	base64Key, err := GetLocalKey(emailAddress, suffix)
	if err != nil {
		return "", keyBytes, err
	}
	keyBytes, err = crypto.DecodeBase64Key32(base64Key)
	if err != nil {
		return base64Key, keyBytes, err
	}
	return base64Key, keyBytes, nil
}

func GetLocalKey64(emailAddress, suffix string) (string, [64]byte, error) {
	var keyBytes [64]byte

	base64Key, err := GetLocalKey(emailAddress, suffix)
	if err != nil {
		return "", keyBytes, err
	}
	keyBytes, err = crypto.DecodeBase64Key64(base64Key)
	if err != nil {
		return base64Key, keyBytes, err
	}
	return base64Key, keyBytes, nil
}

func GetLocalKey(emailAddress, suffix string) (string, error) {
	if !address.ValidEmailAddress(emailAddress) {
		return "", errors.New("malformed email address")
	}

	homePath, err := storage.LocalHomePath(emailAddress)
	if err != nil {
		return "", err
	}
	keyFilename := strings.ToLower(emailAddress) + "." + suffix
	keyPath := filepath.Join(homePath, keyFilename)

	if _, err := os.Stat(filepath.Dir(keyPath)); os.IsNotExist(err) {
		return "", err
	}

	_, err = os.Stat(keyPath)
	if err != nil {
		return "", err
	}

	key, err := os.ReadFile(keyPath)
	if err != nil {
		return "", err
	}

	return string(key), nil
}
