package crypto

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/nacl/box"
	"golang.org/x/crypto/nacl/sign"
	"io"
	"math/big"
	"os"
)

const CHECKSUM_ALGORITHM = "sha256"
const SIGNING_ALGORITHM = "ed25519"
const ANONYMOUS_ENCRYPTION_CIPHER = "curve25519xsalsa20poly1305"
const SYMMETRIC_CIPHER = "xchacha20poly1305"
const SYMMETRIC_FILE_CIPHER = "secretstream_xchacha20poly1305"

func init() {
	assertAvailablePRNG()
}

func assertAvailablePRNG() {
	// Assert that a cryptographically secure PRNG is available.
	// Panic otherwise.
	buf := make([]byte, 1)

	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		panic(fmt.Sprintf("crypto/rand is unavailable: Read() failed with %#v", err))
	}
}

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
// Adapted from https://elithrar.github.io/article/generating-secure-random-numbers-crypto-rand/
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

// GenerateRandomString returns a securely generated random string.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomString(n int) (string, error) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		ret[i] = letters[num.Int64()]
	}

	return string(ret), nil
}

func GenerateRandomToken(tokenLength int) (string, error) {
	authToken, err := GenerateRandomString(tokenLength)
	if err != nil {
		return "", err
	}
	return authToken, nil
}

func DecodeBase64Key32(cryptoKey string) ([32]byte, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(cryptoKey)
	if err != nil {
		return [32]byte{}, err
	}
	var key [32]byte
	copy(key[:], keyBytes)
	return key, nil
}

func DecodeBase64Key64(cryptoKey string) ([64]byte, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(cryptoKey)
	if err != nil {
		return [64]byte{}, err
	}
	var key [64]byte
	copy(key[:], keyBytes)
	return key, nil
}

func EncryptAnonymous(publicKey [32]byte, data []byte) (string, error) {
	encryptedData, err := box.SealAnonymous(nil, data, &publicKey, rand.Reader)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encryptedData), nil
}

func DecryptAnonymous(privateKey, publicKey [32]byte, ciphertext string) ([]byte, error) {
	cipherbytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, err
	}

	decryptedMessage, ok := box.OpenAnonymous(nil, cipherbytes, &publicKey, &privateKey)
	if !ok {
		return nil, errors.New("Decryption failed")
	}
	return decryptedMessage, nil
}

func Fingerprint(key []byte) string {
	hexSum, _ := Sha256Hash(key)
	return hexSum
}

func RandomPassword() [64]byte {
	passwordString := [64]byte{}
	_, err := io.ReadFull(rand.Reader, passwordString[:])
	if err != nil {
		panic(err)
	}
	return passwordString
}

func GenerateEncryptionKeys() (privKeyEncodedStr string, pubKeyEncodedStr string) {
	publicKey, privateKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	privKeyEncodedStr = base64.StdEncoding.EncodeToString(privateKey[:])
	pubKeyEncodedStr = base64.StdEncoding.EncodeToString(publicKey[:])
	return
}

func GenerateSigningKeys() (privKeyEncodedStr string, pubKeyEncodedStr string) {
	publicKey, privateKey, err := sign.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	privKeyEncodedStr = base64.StdEncoding.EncodeToString(privateKey[:])
	pubKeyEncodedStr = base64.StdEncoding.EncodeToString(publicKey[:])
	return
}

func SignData(publicKey [32]byte, privateKey [64]byte, data []byte) string {
	signature := sign.Sign(nil, data, &privateKey)
	return base64.StdEncoding.EncodeToString(signature)
}

func VerifySignature(publicKey [32]byte, signatureStr string, origData []byte) bool {
	signature, err := base64.StdEncoding.DecodeString(signatureStr)
	if err != nil {
		fmt.Println("Error decoding signature:", err)
		return false
	}

	verifiedMessage, ok := sign.Open(nil, signature, &publicKey)
	if !ok {
		fmt.Println("Signature verification failed.")
		return false
	}

	return bytes.Equal(origData, verifiedMessage)
}

func Sha256Hash(content []byte) (string, []byte) {
	plainHash := sha256.New()
	plainHash.Write(content)
	sumBytes := plainHash.Sum(nil)
	return hex.EncodeToString(sumBytes), sumBytes
}

func Checksum(content []byte) (string, []byte) {
	hexSum, sumBytes := Sha256Hash(content)
	return hexSum, sumBytes
}

func Xchacha20Poly1305Encrypt(plaintext, key []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aead.NonceSize(), aead.NonceSize()+len(plaintext)+aead.Overhead())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return aead.Seal(nonce, nonce, plaintext, nil), nil
}

func Xchacha20Poly1305Decrypt(ciphertext, key []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, nil
	}

	if len(ciphertext) < aead.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:aead.NonceSize()], ciphertext[aead.NonceSize():]
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

func Xchacha20Poly1305DecodeDecrypt(ciphertext string, key []byte) ([]byte, error) {
	cipherbytes, err := base64.StdEncoding.DecodeString((ciphertext))
	if err != nil {
		return nil, err
	}
	return Xchacha20Poly1305Decrypt(cipherbytes, key)
}

func Xchacha20Poly1305DecryptFile(encryptedFilePath, decryptedFilePath string, key []byte) (*IOInfo, error) {
	plainHash := sha256.New()
	cipherHash := sha256.New()

	encryptedData, err := os.ReadFile(encryptedFilePath)
	if err != nil {
		return nil, err
	}
	cipherHash.Write(encryptedData)

	decryptedData, err := Xchacha20Poly1305Decrypt(encryptedData, key)
	if err != nil {
		return nil, err
	}
	plainHash.Write(decryptedData)

	err = os.WriteFile(decryptedFilePath, decryptedData, 0644)
	if err != nil {
		return nil, err
	}

	cipherChecksum := cipherHash.Sum(nil)
	plainChecksum := plainHash.Sum(nil)
	return &IOInfo{
		InputSize:      int64(len(encryptedData)),
		InputChecksum:  hex.EncodeToString(cipherChecksum),
		OutputSize:     int64(len(decryptedData)),
		OutputChecksum: hex.EncodeToString(plainChecksum),
	}, nil
}
