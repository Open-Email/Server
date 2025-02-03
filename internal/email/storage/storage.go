package storage

import (
	"bufio"
	"email.mercata.com/internal/consts"
	"email.mercata.com/internal/utils"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const MESSAGES_INDEX_FILENAME = "index"

const MESSAGES_STORE_DIRECTORY = "store"

const MESSAGES_ACCESS_LOG_FILENAME = "access"
const MESSAGES_ACCESS_LOG_COLUMN_SEPARATOR = ","

const MESSAGES_LINKS_DIRECTORY = "links"

const STATUS_COLUMN_SEPARATOR = ","
const STATUS_WAIT_TAG = "WAIT"
const STATUS_EXPIRED_TAG = "EXPIRED"

func MessagesStatus(homeDirPath string) ([]string, error) {
	var messageAccessLines []string
	indexFile, err := os.Open(IndexPath(homeDirPath))
	if err != nil {
		if os.IsNotExist(err) {
			// No messages
			return messageAccessLines, nil
		}
		return messageAccessLines, err
	}
	defer indexFile.Close()

	scanner := bufio.NewScanner(indexFile)
	for scanner.Scan() {
		msgStr := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(msgStr, "#") {
			continue
		}
		parts := strings.SplitN(msgStr, MESSAGES_INDEX_COLUMN_SEPARATOR, MESSAGES_INDEX_COLUMNS_COUNT)
		if len(parts) < MESSAGES_INDEX_COLUMNS_COUNT {
			continue
		}
		messageID := parts[MESSAGES_INDEX_COLUMNS_COUNT-1]
		accessPath := AccessLogPath(homeDirPath, messageID)
		messageAccessExists, err := utils.FilePathExists(accessPath)
		if err != nil {
			return messageAccessLines, err
		}

		// messagePath := MessagePath(homeDirPath, messageID)

		//TODO Instead of tag, return date
		expired := false
		// expired, err := utils.IsDirectoryOlderThan(messagePath, consts.MAX_MESSAGE_TIME)
		// if err != nil {
		// 	fmt.Println("file stat error:", messagePath)
		// }

		if messageAccessExists {
			accesses, err := ioutil.ReadFile(accessPath)
			if err != nil {
				return messageAccessLines, err
			}
			for _, accessLine := range strings.Split(string(accesses), "\n") {
				accessLine = strings.TrimSpace(accessLine)
				if (accessLine == "") || strings.HasPrefix(accessLine, "#") {
					continue
				}
				messageAccessLines = append(messageAccessLines, strings.Join([]string{MessageStatusTag(expired), messageID, accessLine}, STATUS_COLUMN_SEPARATOR))
			}
		} else {
			messageAccessLines = append(messageAccessLines, strings.Join([]string{MessageStatusTag(expired), messageID, ""}, STATUS_COLUMN_SEPARATOR))
		}

		if expired {
			err = DeleteMessageDir(homeDirPath, messageID)
			if err != nil {
				fmt.Println("message could not be removed:", messageID)
			}
			err = RemoveMessageFromIndex(homeDirPath, messageID)
			if err != nil {
				fmt.Println("message could not be removed from index:", messageID)
			}
		}
	}
	return messageAccessLines, nil
}

func MessageStatusTag(expired bool) string {
	if expired {
		return STATUS_EXPIRED_TAG
	}
	return STATUS_WAIT_TAG
}

func AccessLogPath(userHomeDirPath, messageID string) string {
	return filepath.Join(userHomeDirPath, MESSAGES_STORE_DIRECTORY, messageID, MESSAGES_ACCESS_LOG_FILENAME)
}

func IndexPath(userHomeDirPath string) string {
	return filepath.Join(userHomeDirPath, MESSAGES_STORE_DIRECTORY, MESSAGES_INDEX_FILENAME)
}

func LogMessageAccess(userHomeDirPath, messageID, link string) error {
	// Messages access file format:
	//
	// 	MessageID, Link, Date(RFC3339)
	//
	accessPath := AccessLogPath(userHomeDirPath, messageID)
	accessLine := link + MESSAGES_ACCESS_LOG_COLUMN_SEPARATOR + utils.ToRFC3339String(utils.TimestampNow())

	alreadyAccessed, err := utils.PrefixExistsInFile(link, accessPath)
	if err != nil {
		return err
	}
	if alreadyAccessed {
		return nil
	}
	err = utils.AppendStringToFile(accessLine, accessPath)
	if err != nil {
		return err
	}
	return nil
}

func MessageExists(userHomeDirPath, messageID string) (string, bool, error) {
	messagePath := filepath.Join(userHomeDirPath, MESSAGES_STORE_DIRECTORY, messageID)
	messageExists, err := utils.FilePathExists(messagePath)
	if err != nil {
		return messagePath, false, err
	}
	if messageExists {
		return messagePath, true, nil
	}
	return messagePath, false, nil
}

func CreateMessageDir(userHomeDirPath, messageID string) (string, error) {
	messagePath := MessagePath(userHomeDirPath, messageID)
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

func DeleteMessageDir(userHomeDirPath, messageID string) error {
	messagePath := MessagePath(userHomeDirPath, messageID)
	_, err := os.Stat(messagePath)
	if err != nil {
		return err
	}
	err = os.RemoveAll(messagePath)
	if err != nil {
		return err
	}
	return nil
}

func MessagesPath(userHomeDirPath string) string {
	return filepath.Join(userHomeDirPath, MESSAGES_STORE_DIRECTORY)
}
func MessagePath(userHomeDirPath, messageID string) string {
	return filepath.Join(MessagesPath(userHomeDirPath), messageID)
}
func MessageEnvelopePath(userHomeDirPath, messageID string) string {
	return filepath.Join(MessagePath(userHomeDirPath, messageID), consts.MESSAGE_DIR_ENVELOPE_FILE_NAME)
}
func MessagePayloadPath(userHomeDirPath, messageID string) string {
	return filepath.Join(MessagePath(userHomeDirPath, messageID), consts.MESSAGE_DIR_PAYLOAD_FILE_NAME)
}
