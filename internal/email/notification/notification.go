package notification

import (
	"email.mercata.com/internal/consts"
	"email.mercata.com/internal/crypto"
	"email.mercata.com/internal/utils"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const NOTIFICATIONS_DIR = "notifications"
const NOTIFICATIONS_COLUMN_SEPARATOR = ","
const NOTIFICATIONS_ID_LENGTH = 32

func Exists(homeDirPath, link string) (bool, error) {
	notificationPath := filepath.Join(homeDirPath, NOTIFICATIONS_DIR, link)
	linkExists, err := utils.FilePathExists(notificationPath)
	if err != nil {
		return false, err
	}
	if linkExists {
		return true, nil
	}
	return false, nil
}

func Store(homeDirPath, link, notifier, notifierSignKey, readerPubEncryptKey string) error {
	err := createNotificationsDir(homeDirPath)
	if err != nil {
		return err
	}

	// Generate the notification identifier
	randomStr, err := crypto.GenerateRandomString(NOTIFICATIONS_ID_LENGTH)
	if err != nil {
		return err
	}

	notificationLine := strings.Join([]string{randomStr, notifier, notifierSignKey, readerPubEncryptKey}, NOTIFICATIONS_COLUMN_SEPARATOR)
	notificationPath := filepath.Join(homeDirPath, NOTIFICATIONS_DIR, link)
	err = os.WriteFile(notificationPath, []byte(notificationLine), 0644)
	if err != nil {
		return err
	}

	go cleanupOldNotifications(homeDirPath)

	return nil
}

func ListAll(homeDirPath string) ([]string, error) {
	var result []string

	notificationsPath := filepath.Join(homeDirPath, NOTIFICATIONS_DIR)

	notificationsPathExists, err := utils.FilePathExists(notificationsPath)
	if err != nil {
		return nil, err
	}
	if !notificationsPathExists {
		return result, nil
	}

	fileInfos, err := ioutil.ReadDir(notificationsPath)
	if err != nil {
		return nil, err
	}

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			continue // Skip directories
		}
		filePath := filepath.Join(notificationsPath, fileInfo.Name())
		fileContents, err := ioutil.ReadFile(filePath)
		if err != nil {
			continue
		}
		result = append(result, fmt.Sprintf("%s%s%s", fileInfo.Name(), NOTIFICATIONS_COLUMN_SEPARATOR, fileContents))
	}
	return result, nil
}

func cleanupOldNotifications(homeDirPath string) {
	notificationsPath := filepath.Join(homeDirPath, NOTIFICATIONS_DIR)
	maxNotificationTime := time.Now().Add(-1 * consts.MAX_NOTIFICATION_TIME)
	err := utils.DeleteFilesOlderThan(notificationsPath, maxNotificationTime)
	if err != nil {
		fmt.Printf("Error cleaning up all notifications: %s\n", err)
	}
}

func createNotificationsDir(homeDirPath string) error {
	path := filepath.Join(homeDirPath, NOTIFICATIONS_DIR)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}
