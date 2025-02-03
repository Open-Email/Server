package user

import (
	"email.mercata.com/internal/crypto"
	"email.mercata.com/internal/email/address"
	"email.mercata.com/internal/email/keys"
	linksPkg "email.mercata.com/internal/email/links"
)

type User struct {
	Address   string
	Domain    string
	LocalPart string

	PublicEncryptionKeyFingerprint string
	PublicSigningKeyFingerprint    string

	PublicEncryptionKeyBase64 string
	PublicEncryptionKey       [32]byte
	PublicSigningKeyBase64    string
	PublicSigningKey          [32]byte

	PrivateEncryptionKeyBase64 string
	PrivateEncryptionKey       [32]byte
	PrivateSigningKeyBase64    string
	PrivateSigningKey          [64]byte
}

type Reader struct {
	User
	Link      string
	SealedKey string
}

func SelfLink(user, domain string) string {
	address := user + "@" + domain
	return linksPkg.Make(address, address)
}

func LocalUser(emailAddress string) (*User, error) {
	user := User{}
	user.Address, user.Domain, user.LocalPart = address.ParseEmailAddress(emailAddress)
	err := getLocalEncryptionKeys(&user)
	if err != nil {
		return nil, err
	}
	err = getLocalSigningKeys(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func AsReader(user *User) *Reader {
	var reader Reader
	reader.Link = linksPkg.Make(user.Address, user.Address)
	reader.User = *user
	return &reader
}

func getLocalEncryptionKeys(user *User) (err error) {
	user.PrivateEncryptionKeyBase64, user.PrivateEncryptionKey, err = keys.GetLocalEncryptionPrivateKey(user.Address)
	if err != nil {
		return err
	}
	user.PublicEncryptionKeyBase64, user.PublicEncryptionKey, err = keys.GetLocalEncryptionPublicKey(user.Address)
	if err != nil {
		return err
	}
	user.PublicEncryptionKeyFingerprint = crypto.Fingerprint(user.PublicEncryptionKey[:])
	return nil
}

func getLocalSigningKeys(user *User) (err error) {
	user.PrivateSigningKeyBase64, user.PrivateSigningKey, err = keys.GetLocalSigningPrivateKey(user.Address)
	if err != nil {
		return err
	}
	user.PublicSigningKeyBase64, user.PublicSigningKey, err = keys.GetLocalSigningPublicKey(user.Address)
	if err != nil {
		return err
	}
	user.PublicSigningKeyFingerprint = crypto.Fingerprint(user.PublicSigningKey[:])
	return nil
}
