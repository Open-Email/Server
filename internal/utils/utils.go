package utils

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func ByteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}

func StringIsAlphaNumeric(str string) bool {
	var r = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	return r.MatchString(str)
}

func CheckStringInFile(filepath, searchStr string) (bool, error) {
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return false, nil
	}

	file, err := os.Open(filepath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), searchStr) {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return false, nil
}

func ListIsUnique(input []string) bool {
	seen := make(map[string]bool)

	for _, item := range input {
		if seen[item] {
			return false
		}
		seen[item] = true
	}

	return true
}

func ListContains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
