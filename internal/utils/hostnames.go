package utils

import (
	"fmt"
	"net"
	"strings"
)

func FilterValidHostnames(lines []string) []string {
	validHostnames := make([]string, 0)

	for _, line := range lines {
		line = strings.TrimSpace(strings.ToLower(line))
		if line != "" {
			if IsValidHostname(line) {
				validHostnames = append(validHostnames, line)
			} else {
				fmt.Println("Invalid hostname, dropped entry: ", line)
			}
		}
	}

	return validHostnames
}

func IsValidHostname(hostname string) bool {
	_, err := net.LookupHost(hostname)
	return err == nil
}
