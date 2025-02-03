/// TODO, divide server from client storage

package storage

import (
	"bufio"
	"email.mercata.com/internal/utils"
	"os"
	"strings"
	"sync"
)

const MESSAGES_INDEX_COLUMN_SEPARATOR = ","
const MESSAGES_INDEX_COLUMNS_COUNT = 4

var indexFileMutexMap = make(map[string]*sync.Mutex)
var indexFileMutexMapMutex sync.Mutex

func FilterMessagesIndex(homeDirPath, link, signingPublicKeyFingerprint, stream string) ([]string, error) {
	// Messages index file format:
	//
	// 	Link, Fingerprint, (StreamID:)MessageID
	//
	messagesIndexPath := IndexPath(homeDirPath)
	var filteredMessageIDs []string

	prefix := strings.Join([]string{
		link,
		signingPublicKeyFingerprint,
		stream,
		"",
	}, MESSAGES_INDEX_COLUMN_SEPARATOR)

	messageIDs, err := utils.FilterFileByPrefix(messagesIndexPath, prefix)
	if err != nil {
		return filteredMessageIDs, err
	}
	for _, messageID := range *messageIDs {
		_, messageExists, err := MessageExists(homeDirPath, messageID)
		if err != nil {
			return filteredMessageIDs, err
		}
		if !messageExists {
			continue
		}
		filteredMessageIDs = append(filteredMessageIDs, messageID)
	}

	return filteredMessageIDs, nil
}

func getIndexFileMutex(filePath string) *sync.Mutex {
	indexFileMutexMapMutex.Lock()
	defer indexFileMutexMapMutex.Unlock()

	mutex, exists := indexFileMutexMap[filePath]
	if !exists {
		mutex = &sync.Mutex{}
		indexFileMutexMap[filePath] = mutex
	}

	return mutex
}

func removeStaleIndexFileMutex(filePath string) {
	indexFileMutexMapMutex.Lock()
	defer indexFileMutexMapMutex.Unlock()

	if _, exists := indexFileMutexMap[filePath]; exists {
		delete(indexFileMutexMap, filePath)
	}
}

func WriteMessageIndex(homeDirPath, link, fingerprint, stream, messageID string) error {
	mutex := getIndexFileMutex(homeDirPath)
	mutex.Lock()
	defer func() {
		mutex.Unlock()
		removeStaleIndexFileMutex(homeDirPath)
	}()

	messagesIndexPath := IndexPath(homeDirPath)
	indexLine := strings.Join([]string{
		link,
		fingerprint,
		stream,
		messageID, // It is important that messageID is last
	}, MESSAGES_INDEX_COLUMN_SEPARATOR)

	exists, err := utils.PrefixExistsInFile(indexLine, messagesIndexPath)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	err = utils.AppendStringToFile(indexLine, messagesIndexPath)
	if err != nil {
		return err
	}
	return nil
}

func RemoveMessageFromIndex(homeDirPath, messageID string) error {
	mutex := getIndexFileMutex(homeDirPath)
	mutex.Lock()
	defer func() {
		mutex.Unlock()
		removeStaleIndexFileMutex(homeDirPath)
	}()

	messagesIndexPath := IndexPath(homeDirPath)
	input, err := os.Open(messagesIndexPath)
	if err != nil {
		return err
	}
	defer input.Close()

	tempOutputPath := messagesIndexPath + "~"
	output, err := os.Create(tempOutputPath)
	if err != nil {
		return err
	}
	defer output.Close()

	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasSuffix(line, MESSAGES_INDEX_COLUMN_SEPARATOR+messageID) {
			_, _ = output.WriteString(line + "\n")
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	err = os.Rename(tempOutputPath, messagesIndexPath)
	if err != nil {
		return err
	}

	return nil
}
