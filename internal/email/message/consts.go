package message

const READER_KEY_LENGTH = 32

const MESSAGE_DIR_BODY_FILE_NAME = "body"
const MESSAGE_DIR_FILE_NAME = "file"
const MESSAGE_DIR_HEADERS_FILE_NAME = "headers"

const HEADER_MESSAGE_ID = "message-id"
const HEADER_MESSAGE_STREAM = "message-stream"
const HEADER_MESSAGE_ACCESS = "message-access"
const HEADER_MESSAGE_CONTENT_HEADERS = "message-headers"
const HEADER_MESSAGE_ENVELOPE_CHECKSUM = "message-checksum"
const HEADER_MESSAGE_ENVELOPE_SIGNATURE = "message-signature"
const HEADER_MESSAGE_ENCRYPTION = "message-encryption"

var PERMITTED_ENVELOPE_KEYS = []string{
	HEADER_MESSAGE_ID,
	HEADER_MESSAGE_STREAM,
	HEADER_MESSAGE_ACCESS,
	HEADER_MESSAGE_CONTENT_HEADERS,
	HEADER_MESSAGE_ENVELOPE_CHECKSUM,
	HEADER_MESSAGE_ENVELOPE_SIGNATURE,
	HEADER_MESSAGE_ENCRYPTION,
}

const HEADER_CONTENT_MESSAGE_ID = "id"
const HEADER_CONTENT_AUTHOR = "author"
const HEADER_CONTENT_DATE = "date"
const HEADER_CONTENT_SIZE = "size"
const HEADER_CONTENT_CHECKSUM = "checksum"
const HEADER_CONTENT_FILE = "file"
const HEADER_CONTENT_SUBJECT = "subject"
const HEADER_CONTENT_SUBJECT_ID = "subject-id"
const HEADER_CONTENT_PARENT_MESSAGE_ID = "parent-message-id"
const HEADER_CONTENT_CATEGORY = "category"
const HEADER_CONTENT_READERS = "readers"

const HEADER_YES_VALUE = "Yes"
const HEADER_NO_VALUE = "No"
const HEADER_ACCESS_LIST_SEPARATOR = ","
const HEADER_ACCESS_FIELDS_SEPARATOR = ":"
const HEADER_KEY_VALUE_SEPARATOR = ":"
const HEADER_CONTENT_READERS_ADDRESS_SEPARATOR = ","
