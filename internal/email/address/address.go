package address

import (
	"regexp"
	"strings"
)

const EmailAddressRegex = `^(?i)(?:[a-z0-9])(?:[a-z0-9\.\-_\+])*@[a-z0-9.-]+\.(?:[a-z]{2,}|xn--[a-z0-9]{2,})$`

/* Intentionally there is no support for "NAME <address>".
 * The name *must* come from the profile.
 *********************************************************/
func ParseEmailAddress(address string) (safeAddress, domain, localPart string) {
	userParts := strings.Split(strings.ToLower(address), "@")
	localPart = strings.TrimSpace(userParts[0])
	domain = strings.TrimSpace(userParts[1])
	return JoinAddress(domain, localPart), domain, localPart
}

func JoinAddress(domain, localPart string) string {
	return localPart + "@" + domain
}

func ValidEmailAddress(value string) bool {
	if len(value) == 0 {
		return false
	}
	emailRegexp := regexp.MustCompile(EmailAddressRegex)
	return emailRegexp.MatchString(value)
}
