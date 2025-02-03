package storage

import (
	"email.mercata.com/internal/consts"
	"email.mercata.com/internal/email/address"
	"email.mercata.com/internal/utils"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const LOCAL_EMAIL_DIRECTORY = ".email2"
const LOCAL_STORAGE_DIRECTORY = "store"
const LOCAL_TEMP_STORAGE_DIRECTORY = ".tmpstore"
const LOCAL_STORAGE_INDEX_FIELD_SEPARATOR = ","

const LOCAL_MESSAGE_READ_FILENAME = "read"
const LOCAL_MESSAGE_BODY_FILENAME = "body.txt"
const LOCAL_MESSAGE_HEADERS_FILENAME = "headers.txt"

func LocalTempMessageExists(authorEmailAddress, messageID string) (string, bool, error) {
	messagePath, err := LocalTempMessagePath(authorEmailAddress, messageID)
	messageExists, err := utils.FilePathExists(messagePath)
	if err != nil {
		return messagePath, false, err
	}
	if messageExists {
		return messagePath, true, nil
	}
	return messagePath, false, nil
}

func LocalMessageExists(authorEmailAddress, folderName, messageID string) (string, bool, error) {
	messagePath, err := LocalMessagePath(authorEmailAddress, folderName, messageID)
	messageExists, err := utils.FilePathExists(messagePath)
	if err != nil {
		return messagePath, false, err
	}
	if messageExists {
		return messagePath, true, nil
	}
	return messagePath, false, nil
}

func LocalMessageRead(authorEmailAddress, folderName, messageID string) (bool, error) {
	messageReadPath, err := LocalMessageReadPath(authorEmailAddress, folderName, messageID)
	messageReadExists, err := utils.FilePathExists(messageReadPath)
	if err != nil {
		return false, err
	}
	if messageReadExists {
		return true, nil
	}
	return false, nil
}

func MarkMessageRead(authorEmailAddress, folderName, messageID string) error {
	isRead, err := LocalMessageRead(authorEmailAddress, folderName, messageID)
	if err != nil {
		return err
	}
	if isRead {
		return nil
	}
	messageReadPath, err := LocalMessageReadPath(authorEmailAddress, folderName, messageID)
	err = os.WriteFile(messageReadPath, []byte{}, 0644)
	if err != nil {
		return err
	}
	return nil
}

func LocalHomePath(userEmailAddress string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	address, _, _ := address.ParseEmailAddress(userEmailAddress)
	return filepath.Join(homeDir, LOCAL_EMAIL_DIRECTORY, address), nil
}

func LocalStorePath(authorEmailAddress string) (string, error) {
	localHomeDir, err := LocalHomePath(authorEmailAddress)
	if err != nil {
		return "", err
	}
	return filepath.Join(localHomeDir, LOCAL_STORAGE_DIRECTORY), nil
}

func LocalFolderPath(userEmailAddress, folderName string) (string, error) {
	homeDir, err := LocalStorePath(userEmailAddress)
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, strings.ToLower(strings.TrimSpace(folderName))), nil
}

func LocalMessagePath(authorEmailAddress, folderName, messageID string) (string, error) {
	folderPath, err := LocalFolderPath(authorEmailAddress, folderName)
	if err != nil {
		return "", err
	}
	return filepath.Join(folderPath, messageID), nil
}

func CreateLocalTempMessageDir(authorEmailAddress, messageID string) (string, error) {
	messagePath, err := LocalTempMessagePath(authorEmailAddress, messageID)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(messagePath); os.IsNotExist(err) {
		err = os.MkdirAll(messagePath, 0755)
		if err != nil {
			return messagePath, err
		}
	} else {
		return messagePath, errors.New("Fatal error: message with same ID present")
	}
	return messagePath, nil
}

func CreateLocalMessageDir(authorEmailAddress, topicID, messageID string) (string, error) {
	messagePath, err := LocalMessagePath(authorEmailAddress, topicID, messageID)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(messagePath); os.IsNotExist(err) {
		err = os.MkdirAll(messagePath, 0755)
		if err != nil {
			return messagePath, err
		}
	} else {
		return messagePath, errors.New("Fatal error: message with same ID present")
	}
	return messagePath, nil
}

func CreateEmailHomePath(emailAddress string) error {
	homeDir, err := LocalHomePath(emailAddress)
	if err != nil {
		return err
	}

	if _, err := os.Stat(homeDir); os.IsNotExist(err) {
		err = os.Mkdir(homeDir, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

func LocalUserTempStorePath(authorEmailAddress string) (string, error) {
	localHomeDir, err := LocalHomePath(authorEmailAddress)
	if err != nil {
		return "", err
	}
	return filepath.Join(localHomeDir, LOCAL_TEMP_STORAGE_DIRECTORY), nil
}

func LocalTempMessagePath(authorEmailAddress, messageID string) (string, error) {
	tmpStoragePath, err := LocalUserTempStorePath(authorEmailAddress)
	if err != nil {
		return "", err
	}
	return filepath.Join(tmpStoragePath, messageID), nil
}

func LocalTempMessageEnvelopePath(authorEmailAddress, messageID string) (string, error) {
	p, err := LocalTempMessagePath(authorEmailAddress, messageID)
	if err != nil {
		return "", err
	}
	return filepath.Join(p, consts.MESSAGE_DIR_ENVELOPE_FILE_NAME), nil
}

func LocalTempMessagePayloadPath(authorEmailAddress, messageID string) (string, error) {
	p, err := LocalTempMessagePath(authorEmailAddress, messageID)
	if err != nil {
		return "", err
	}
	return filepath.Join(p, consts.MESSAGE_DIR_PAYLOAD_FILE_NAME), nil
}

func LocalMessageHeadersPath(authorEmailAddress, folderName, messageID string) (string, error) {
	p, err := LocalMessagePath(authorEmailAddress, folderName, messageID)
	if err != nil {
		return "", err
	}
	return filepath.Join(p, LOCAL_MESSAGE_HEADERS_FILENAME), nil
}

func LocalMessageBodyPath(authorEmailAddress, folderName, messageID string) (string, error) {
	p, err := LocalMessagePath(authorEmailAddress, folderName, messageID)
	if err != nil {
		return "", err
	}
	return filepath.Join(p, LOCAL_MESSAGE_BODY_FILENAME), nil
}

func LocalMessageFileNamePath(authorEmailAddress, folderName, messageID, fileName string) (string, error) {
	p, err := LocalMessagePath(authorEmailAddress, folderName, messageID)
	if err != nil {
		return "", err
	}
	return filepath.Join(p, fileName), nil
}

func LocalMessageReadPath(authorEmailAddress, folderName, messageID string) (string, error) {
	p, err := LocalMessagePath(authorEmailAddress, folderName, messageID)
	if err != nil {
		return "", err
	}
	return filepath.Join(p, LOCAL_MESSAGE_READ_FILENAME), nil
}

// func LocalTopicPath(userAddress, topicID string) (string, error) {
// 	storePath, err := LocalUserStorePath(userAddress)
// 	if err != nil {
// 		return "", err
// 	}
// 	return filepath.Join(storePath, topicID), nil
// }

// func AddToLocalTopicsIndex(userAddress, topicID, dateStamp string) error {
// 	topicsFilePath := LocalTopicsIndexPath(userAddress)
// 	exists, err := utils.PrefixExistsInFile(topicID, topicsFilePath)
// 	if err != nil {
// 		return err
// 	}
// 	if exists {
// 		return nil
// 	}
// 	line := strings.Join([]strings{topicID, time.Now().Unix()}, LOCAL_STORAGE_INDEX_FIELD_SEPARATOR)
// 	err = utils.AppendStringToFile(line, topicsFilePath)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// List all folders of messages, regardless if opened or not.
func ListLocalMessageIDs(userAddress, folderName string) ([]string, error) {
	folderPath, err := LocalFolderPath(userAddress, folderName)
	if err != nil {
		return nil, err
	}
	messageIds, err := utils.ListDirectories(folderPath)
	if err != nil {
		return nil, err
	}
	var filteredMessageIDs []string
	for _, msgId := range messageIds {
		if utils.ValidMessageID(msgId) {
			filteredMessageIDs = append(filteredMessageIDs, msgId)
		}
	}
	return filteredMessageIDs, nil
}

// func LastTopicMessage(userAddress, topicID string) (string, error) {
// 	topicPath, err := LocalTopicPath(userAddress, topicID)
// 	if err != nil {
// 		return "", err
// 	}
// 	messageId, err := utils.LatestDirectory(topicPath)
// 	if err != nil {
// 		return "", err
// 	}
// 	return messageId, nil
// }

// func ListTopicMessages(userAddress, topicID string) ([]string, error) {
// 	topicPath, err := LocalTopicPath(userAddress, topicID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	messageIds, err := utils.ListDirectories(topicPath)
// 	if err != nil {
// 		return nil, err
// 	}
// 	// TODO: check if directories are valid messageIDs
// 	return messageIds, nil
// }
