package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type CommandFunc func([]string)

var commandMap = map[string]CommandFunc{
	"keys-gen":    keysGenCommand,
	"keys-lookup": keysLookupCommand,

	"links-make":   linksMakeCommand,
	"links-list":   linksListCommand,
	"links-store":  linksStoreCommand,
	"links-delete": linksDeleteCommand,

	"notifications-store": notificationsStoreCommand,
	"notifications-list":  notificationsListCommand,

	"messages-open":   messagesOpenCommand,
	"messages-author": messagesAuthorCommand,

	"messages-list":   messagesListCommand,
	"messages-fetch":  messagesFetchCommand,
	"messages-status": messagesStatusCommand,
	"messages-store":  messagesStoreCommand,
	"messages-delete": messagesDeleteCommand,

	"profile-fetch":       profileFetchCommand,
	"profile-store":       profileStoreCommand,
	"profile-image-fetch": profileImageFetchCommand,
	"profile-image-store": profileImageStoreCommand,

	"provision": provisionCommand,
}

func main() {
	fs := flag.NewFlagSet("api-client", flag.ExitOnError)
	fs.Parse(os.Args[1:])

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <flag-set>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Available flag sets: %s\n", strings.Join(getAvailableCommands(), ", "))
		return
	}

	if len(fs.Args()) > 0 {
		cmdName := fs.Arg(0)
		if cmdFunc, found := commandMap[cmdName]; found {
			cmdFunc(fs.Args()[1:])
		} else {
			fmt.Fprintf(os.Stderr, "Unknown flag set: %s\n", os.Args[1])
		}
	}
}

func getAvailableCommands() []string {
	var available []string
	for cmd := range commandMap {
		available = append(available, cmd)
	}
	return available
}
