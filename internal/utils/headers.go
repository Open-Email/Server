package utils

import (
	"strings"
)

func ParseHeadersAttributes(value string) map[string]string {
	headerAttrsMap := make(map[string]string)
	pairs := strings.Split(value, ";")
	for _, kv := range pairs {
		parts := strings.SplitN(kv, "=", 2)
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		headerAttrsMap[key] = strings.TrimSpace(parts[1])
	}
	return headerAttrsMap
}
