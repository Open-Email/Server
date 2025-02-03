package utils

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func FilePathExists(filePath string) (bool, error) {
	_, err := os.Stat(filePath)
	if err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

func DetermineFileType(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return "", err
	}

	return DetermineFileTypeOfData(&buffer)
}

func DetermineFileTypeOfData(data *[]byte) (string, error) {
	contentType := http.DetectContentType((*data)[:512])
	parts := strings.Split(contentType, ";")
	return parts[0], nil
}

type FileInfo struct {
	Path       string
	BaseName   string
	Type       string
	Size       int64
	ModifiedAt time.Time
}

func GetFileInfo(filePath string) (*FileInfo, error) {
	absoluteFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	fstat, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	detectedFileType, err := DetermineFileType(filePath)
	if err != nil {
		detectedFileType = "application/octet-stream"
	}

	return &FileInfo{
		Path:       absoluteFilePath,
		BaseName:   filepath.Base(absoluteFilePath),
		ModifiedAt: fstat.ModTime().UTC(),
		Size:       fstat.Size(),
		Type:       detectedFileType,
	}, nil
}

func PrefixExistsInFile(targetString, filePath string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), targetString) {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return false, nil
}

func FilterFileByPrefix(filePath, targetString string) (*[]string, error) {
	var results []string

	targetString = strings.TrimSpace(targetString)
	if targetString == "" {
		return &results, nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &results, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, targetString) {
			remainder := strings.TrimSpace(line[len(targetString):])
			if remainder == "" {
				continue
			}
			results = append(results, remainder)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &results, nil
}

func DeleteFilesExcept(path, prefixToDelete string, exceptions []string) error {
	pattern := filepath.Join(path, prefixToDelete+"*")
	matchingFiles, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	for _, file := range matchingFiles {
		fileName := filepath.Base(file)

		if !ListContains(exceptions, fileName) {
			if err := os.Remove(file); err != nil {
				return err
			}
			fmt.Println("Deleted:", file)
		}
	}

	return nil
}

func DeleteFilesOlderThan(folderPath string, ago time.Time) error {
	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && info.ModTime().Before(ago) {
			fmt.Println("Removing:", path)
			if err := os.Remove(path); err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func AppendStringToFile(content, filePath string) error {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	_, err = file.WriteString(content)
	if err != nil {
		return err
	}

	return nil
}

func DirectorySize(dirPath string) (int64, error) {
	var size int64

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0, err
	}

	return size, nil
}

func IsDirectoryOlderThan(path string, maxAge time.Duration) (bool, error) {
	currentTime := time.Now()

	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	if !info.IsDir() {
		return false, nil
	}

	if info.ModTime().Before(currentTime.Add(-maxAge)) {
		return true, nil
	}

	return false, nil
}

func ListDirectories(path string) ([]string, error) {
	dir, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	entries, err := dir.Readdir(-1)
	if err != nil {
		return nil, err
	}

	var dirNames []string

	for _, entry := range entries {
		if entry.IsDir() {
			dirNames = append(dirNames, entry.Name())
		}
	}

	sort.Slice(dirNames, func(i, j int) bool {
		pathI := filepath.Join(path, dirNames[i])
		pathJ := filepath.Join(path, dirNames[j])

		infoI, _ := os.Stat(pathI)
		infoJ, _ := os.Stat(pathJ)

		return infoI.ModTime().After(infoJ.ModTime())
	})

	return dirNames, nil
}

func LatestDirectory(path string) (string, error) {
	var latestDir string
	var latestModTime time.Time

	// Walk through the directory and its subdirectories
	err := filepath.Walk(path, func(currentPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err // Handle error, if any
		}

		// Check if it's a directory and if the modification time is later than the latest
		if info.IsDir() && info.ModTime().After(latestModTime) {
			latestModTime = info.ModTime()
			latestDir = info.Name()
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return latestDir, nil
}

const BUFFER_SIZE = 8192

func CopyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file.", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	if err != nil {
		panic(err)
	}

	buf := make([]byte, BUFFER_SIZE)
	for {
		n, err := source.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		if _, err := destination.Write(buf[:n]); err != nil {
			return err
		}
	}
	return err
}
