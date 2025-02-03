package nonce

import (
	"email.mercata.com/internal/crypto"
	"email.mercata.com/internal/email/user"
	"email.mercata.com/internal/utils"
	"encoding/base64"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

const NONCES_FILENAME = ".nonces_"
const NONCES_FILENAME_DATE = "2006-01-02"
const NONCES_COLUMN_SEPARATOR = ","

const NONCE_SCHEME = "SOTN"
const NONCE_HEADER_ATTRIBUTE_VALUE = "nonce"
const NONCE_HEADER_ATTRIBUTE_ALGORITHM = "algorithm"
const NONCE_HEADER_ATTRIBUTE_SIGNATURE = "signature"
const NONCE_HEADER_ATTRIBUTE_KEY = "key"

var ErrorNonceReplay = errors.New("nonce replay")
var ErrorBadNonceHeader = errors.New("bad nonce header")
var ErrorBadNonceSignature = errors.New("bad nonce signature")
var ErrorBadNonceKey = errors.New("bad nonce signing key")

type Nonce struct {
	Value      string
	Signature  string
	SigningKey [32]byte
	Date       string

	// Verification use only
	SigningKeyBase64      string
	SigningKeyFingerprint string
	SigningAlgorithm      string
}

func ForUser(user *user.User) (*Nonce, error) {
	return New(user.PublicSigningKey, user.PrivateSigningKey)
}

func New(publicKey [32]byte, privateKey [64]byte) (*Nonce, error) {
	val, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return nil, err
	}

	signedNonce := crypto.SignData(publicKey, privateKey, []byte(val))

	return &Nonce{
		SigningAlgorithm: crypto.SIGNING_ALGORITHM,
		Value:            val,
		Signature:        signedNonce,
		SigningKeyBase64: base64.StdEncoding.EncodeToString(publicKey[:]),
		Date:             utils.ToRFC3339String(utils.TimestampNow()),
	}, nil
}

func ToHeader(nonce *Nonce) string {
	return NONCE_SCHEME + " " + strings.Join([]string{
		strings.Join([]string{NONCE_HEADER_ATTRIBUTE_VALUE, nonce.Value}, "="),
		strings.Join([]string{NONCE_HEADER_ATTRIBUTE_ALGORITHM, nonce.SigningAlgorithm}, "="),
		strings.Join([]string{NONCE_HEADER_ATTRIBUTE_SIGNATURE, nonce.Signature}, "="),
		strings.Join([]string{NONCE_HEADER_ATTRIBUTE_KEY, nonce.SigningKeyBase64}, "="),
	}, ", ")
}

func FromHeader(nonceHeader string) (*Nonce, error) {
	values := strings.SplitN(nonceHeader, NONCE_SCHEME, 2)
	trimmedValues := strings.TrimSpace(values[1])
	trimmedValues = strings.ReplaceAll(trimmedValues, "\t", "")
	trimmedValues = strings.ReplaceAll(trimmedValues, "\n", "")
	kvs := strings.Split(trimmedValues, ",")
	nonce := Nonce{Date: utils.ToRFC3339String(utils.TimestampNow())}
	for _, kv := range kvs {
		parts := strings.SplitN(strings.TrimSpace(kv), "=", 2)
		value := parts[1]
		switch strings.ToLower(parts[0]) {
		case NONCE_HEADER_ATTRIBUTE_ALGORITHM:
			nonce.SigningAlgorithm = value

		case NONCE_HEADER_ATTRIBUTE_VALUE:
			if nonce.Value != "" {
				return nil, ErrorBadNonceHeader
			}
			nonce.Value = value

		case NONCE_HEADER_ATTRIBUTE_SIGNATURE:
			if nonce.Signature != "" {
				return nil, ErrorBadNonceHeader
			}
			nonce.Signature = value

		case NONCE_HEADER_ATTRIBUTE_KEY:
			if nonce.SigningKeyBase64 != "" {
				return nil, ErrorBadNonceHeader
			}
			pubKey, err := crypto.DecodeBase64Key32(value)
			if err != nil {
				return nil, err
			}
			nonce.SigningKeyBase64 = value
			nonce.SigningKey = pubKey
			nonce.SigningKeyFingerprint = crypto.Fingerprint(pubKey[:])
		}
	}
	if nonce.Value == "" || nonce.Signature == "" || nonce.SigningKeyBase64 == "" {
		return nil, ErrorBadNonceHeader
	}
	return &nonce, nil
}

func VerifySignature(nonce *Nonce) error {
	if crypto.Fingerprint(nonce.SigningKey[:]) != nonce.SigningKeyFingerprint {
		return ErrorBadNonceKey
	}
	if !crypto.VerifySignature(nonce.SigningKey, nonce.Signature, []byte(nonce.Value)) {
		return ErrorBadNonceSignature
	}
	return nil
}

func IsUnique(homeDirPath string, nonce *Nonce) error {
	currentTime := time.Now()
	todaysNoncesFilename := NONCES_FILENAME + currentTime.Format(NONCES_FILENAME_DATE)
	yesterdaysNoncesFilename := NONCES_FILENAME + currentTime.AddDate(0, 0, -1).Format(NONCES_FILENAME_DATE)

	todaysNoncePath := filepath.Join(homeDirPath, todaysNoncesFilename)
	yesterdaysNoncePath := filepath.Join(homeDirPath, yesterdaysNoncesFilename)

	exists, err := utils.PrefixExistsInFile(nonce.Value, todaysNoncePath)
	if err != nil {
		return err
	}
	if exists {
		return ErrorNonceReplay
	}
	exists, err = utils.PrefixExistsInFile(nonce.Value, yesterdaysNoncePath)
	if err != nil {
		return err
	}
	if exists {
		return ErrorNonceReplay
	}
	return nil
}

func Record(homeDirPath string, nonce *Nonce) error {
	currentTime := time.Now()
	todaysNoncesFilename := NONCES_FILENAME + currentTime.Format(NONCES_FILENAME_DATE)
	todaysNoncePath := filepath.Join(homeDirPath, todaysNoncesFilename)

	nonceLine := nonce.Value + NONCES_COLUMN_SEPARATOR + nonce.Date
	err := utils.AppendStringToFile(nonceLine, todaysNoncePath)
	if err != nil {
		return err
	}

	go Cleanup(homeDirPath)

	return nil
}

func Cleanup(homeDirPath string) {
	currentTime := time.Now()
	todaysNoncesFilename := NONCES_FILENAME + currentTime.Format(NONCES_FILENAME_DATE)
	yesterdaysNoncesFilename := NONCES_FILENAME + currentTime.AddDate(0, 0, -1).Format(NONCES_FILENAME_DATE)
	err := utils.DeleteFilesExcept(homeDirPath, NONCES_FILENAME, []string{todaysNoncesFilename, yesterdaysNoncesFilename})
	if err != nil {
		fmt.Printf("Error cleaning up all nonces: %s\n", err)
	}
}
