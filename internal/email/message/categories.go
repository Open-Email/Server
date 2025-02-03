package message

import (
	"email.mercata.com/internal/utils"
)

var CATEGORIES = []string{
	"personal",
	"chat",
	"transitory",
	"notification",
	"transaction",
	"promotion",
	"letter",
	"file",
	"informational",
	"pass",
	"funds",
	"encryption-key",
	"signing-key",
}

const DEFAULT_MESSAGE_CATEGORY = "personal"
const MESSAGE_FILE_CATEGORY = "file"

func ValidCategory(category string) bool {
	return utils.ListContains(CATEGORIES, category)
}
