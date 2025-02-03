package links

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

func Make(emailAddress1, emailAddress2 string) string {
	emailAddress1 = strings.ToLower(strings.TrimSpace(emailAddress1))
	emailAddress2 = strings.ToLower(strings.TrimSpace(emailAddress2))
	strs := []string{emailAddress1, emailAddress2}
	sort.Strings(strs)
	hash := sha256.Sum256([]byte(strings.Join(strs, "")))
	return hex.EncodeToString(hash[:])
}
