package profile

import (
	"bufio"
	"bytes"
	"email.mercata.com/internal/consts"
	"email.mercata.com/internal/crypto"
	addressPkg "email.mercata.com/internal/email/address"
	"email.mercata.com/internal/email/mca"
	"email.mercata.com/internal/email/user"
	"email.mercata.com/internal/utils"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const LOCAL_PROFILE_DIRECTORY = "profile"
const LOCAL_PROFILE_IMAGE_FILENAME = "image"
const LOCAL_PROFILE_DATA_FILENAME = "data"

var PERMITTED_PROFILE_IMAGE_MIME_TYPES = []string{
	"image/webp",
	"image/png",
	"image/jpeg",
}

type Profile struct {
	// Only fields of interest that are needed for authentication
	user.User
	Name string

	IsAway      bool
	AwayMessage string

	PublicAccess           bool
	LastSeenPublicTracking bool

	LastEncryptionKeyFingerprint string
	LastSigningKeyFingerprint    string
	LastEncryptionKeyBase64      string
	LastEncryptionKey            [32]byte
	LastSigningKeyBase64         string
	LastSigningKey               [32]byte

	RemoteBody *[]byte
}

func CreateProfileDir(userHomeDirPath string) error {
	path := GetLocalProfilePath(userHomeDirPath)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetRemoteProfile(address, domain, localPart string) (*Profile, error) {
	var errUnavailable = errors.New("profile unavailable")

	hosts, err := mca.LookupEmailHosts(domain, localPart)
	if err != nil {
		return nil, err
	}
	if len(hosts) == 0 {
		return nil, errUnavailable
	}

	for _, host := range hosts {
		profileURI := fmt.Sprintf("https://%s/%s/%s/%s/profile", host, consts.PUBLIC_API_PATH_PREFIX, domain, localPart)
		resp, err := http.Get(profileURI)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, errUnavailable
		}
		if err == nil {
			body, readErr := io.ReadAll(resp.Body)
			if readErr != nil {
				continue
			}
			profile := Profile{RemoteBody: &body}
			profile.User.Address = address
			profile.User.Domain = domain
			profile.User.LocalPart = localPart

			err = ParseProfile(&profile, body)
			if err != nil {
				return nil, err
			}
			return &profile, nil
		}
	}
	return nil, errUnavailable
}

func GetRemoteProfileImage(domain, localPart string) (*[]byte, error) {
	hosts, err := mca.LookupEmailHosts(domain, localPart)
	if err != nil || len(hosts) == 0 {
		return nil, err
	}

	for _, host := range hosts {
		profileURI := fmt.Sprintf("https://%s/mail/%s/%s/image", host, domain, localPart)
		resp, err := http.Get(profileURI)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, errors.New("profile image unavailable")
		}
		if err == nil {
			body, readErr := io.ReadAll(resp.Body)
			if readErr != nil {
				continue
			}
			return &body, nil
		}
	}
	return nil, nil
}

func GetLocalProfilePath(userHomeDirPath string) string {
	return filepath.Join(userHomeDirPath, LOCAL_PROFILE_DIRECTORY)
}

func GetLocalProfileDataPath(userHomeDirPath string) string {
	return filepath.Join(GetLocalProfilePath(userHomeDirPath), LOCAL_PROFILE_DATA_FILENAME)
}

func GetLocalProfileImagePath(userHomeDirPath string) string {
	return filepath.Join(GetLocalProfilePath(userHomeDirPath), LOCAL_PROFILE_IMAGE_FILENAME)
}

func ImagePathFileTypeIsPermitted(imagePath string) (*string, bool, error) {
	mimeType, err := utils.DetermineFileType(imagePath)
	if err != nil {
		return nil, false, err
	}

	if !ImageMimeTypeIsPermitted(mimeType) {
		return nil, false, nil
	}
	return &mimeType, true, nil
}

func ImageMimeTypeIsPermitted(mimeType string) bool {
	return utils.ListContains(PERMITTED_PROFILE_IMAGE_MIME_TYPES, mimeType)
}

func SetLocalProfileImage(userHomeDirPath string, newProfileData *[]byte) error {
	err := ioutil.WriteFile(GetLocalProfileImagePath(userHomeDirPath), *newProfileData, 0644)
	if err != nil {
		return err
	}
	return nil
}

func SetLocalProfile(userHomeDirPath string, newProfileData *[]byte) error {
	err := CreateProfileDir(userHomeDirPath)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(GetLocalProfileDataPath(userHomeDirPath), *newProfileData, 0644)
	if err != nil {
		return err
	}
	return nil
}

func GetLocalProfile(userHomeDirPath, domain, localPart string) (*Profile, error) {
	profileData, err := os.ReadFile(GetLocalProfileDataPath(userHomeDirPath))
	if err != nil {
		return nil, err
	}
	profile := Profile{RemoteBody: &profileData, PublicAccess: true}
	profile.User.Address = addressPkg.JoinAddress(domain, localPart)
	profile.User.Domain = domain
	profile.User.LocalPart = localPart

	err = ParseProfile(&profile, profileData)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func ParseProfile(p *Profile, body []byte) error {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if err := assignValueToField(p, key, value); err != nil {
				return err
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func assignValueToField(profile *Profile, key, value string) error {
	var err error
	strval := strings.TrimSpace(value)
	if strval == "" {
		return nil
	}

	switch key {
	//
	// TODO: Add all fields
	//
	case PROFILE_FIELD_PUBLIC_ACCESS:
		profile.PublicAccess = (strings.ToLower(value) != "no")

	case PROFILE_FIELD_AWAY:
		profile.IsAway = (strings.ToLower(value) == "yes")

	case PROFILE_FIELD_AWAY_WARNING:
		profile.AwayMessage = value

	case PROFILE_FIELD_NAME:
		profile.Name = value

	case PROFILE_FIELD_ENCRYPTION_KEY:
		profile.User.PublicEncryptionKeyBase64, profile.User.PublicEncryptionKey, profile.User.PublicEncryptionKeyFingerprint, err = extractKeyData(crypto.ANONYMOUS_ENCRYPTION_CIPHER, value)
		if err != nil {
			return err
		}

	case PROFILE_FIELD_SIGNING_KEY:
		profile.User.PublicSigningKeyBase64, profile.User.PublicSigningKey, profile.User.PublicSigningKeyFingerprint, err = extractKeyData(crypto.SIGNING_ALGORITHM, value)
		if err != nil {
			return err
		}

	case PROFILE_FIELD_LAST_SIGNING_KEY:
		profile.LastSigningKeyBase64, profile.LastSigningKey, profile.LastSigningKeyFingerprint, err = extractKeyData(crypto.SIGNING_ALGORITHM, value)
		if err != nil {
			return err
		}

	case PROFILE_FIELD_UPDATED:
		// TODO
	case PROFILE_LAST_SEEN_PUBLIC:
		// TODO Implement tracking
		profile.LastSeenPublicTracking = (strings.ToLower(value) != "no")
	}
	return nil
}

func extractKeyData(algorithm string, strkey string) (string, [32]byte, string, error) {
	var key [32]byte

	valuesMap := utils.ParseHeadersAttributes(strkey)
	attrAlgorithm, ok := valuesMap["algorithm"]
	if !ok {
		return strkey, key, "", errors.New("'algorithm' attribute not present in profile Key data")
	}
	if strings.ToLower(attrAlgorithm) != algorithm {
		return strkey, key, "", errors.New("algorithm mismatch in Key data")
	}
	value, ok := valuesMap["value"]
	if !ok {
		return strkey, key, "", errors.New("'value' attribute not present in profile Key data")
	}
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return strkey, key, "", err
	}
	return value, key, crypto.Fingerprint(data[:]), nil
}

func IsFunctionalProfile(p *Profile) bool {
	return p.PublicSigningKeyBase64 != ""
}
