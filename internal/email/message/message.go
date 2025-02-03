package message

import (
	"email.mercata.com/internal/crypto"
	"email.mercata.com/internal/email/address"
	linksPkg "email.mercata.com/internal/email/links"
	"email.mercata.com/internal/email/user"
	"email.mercata.com/internal/utils"
	"strings"
	"time"
)

const MESSAGE_STREAM_MAXIMUM_LENGTH = 128
const MESSAGE_SUBJECT_MAXIMUM_LENGTH = 128

type Message struct {
	ID          string
	Author      user.User
	IsBroadcast bool
	AccessKey   []byte
	Readers     []user.Reader

	StreamID string

	PayloadCipher *crypto.CipherInfo

	EnvelopeHeadersOrder             string
	EnvelopeHeadersChecksum          string
	EnvelopeHeadersChecksumUnparsed  string
	EnvelopeHeadersSignature         string
	EnvelopeHeadersSignatureUnparsed string

	Content

	// Verification fields
	AccessList          string
	ContentHeadersData  string
	ContentHeadersBytes []byte

	// Raw envelope headers
	EnvelopeHeadersList []string

	//
	IsRead bool
}

type Content struct {
	Date             time.Time
	Subject          string
	SubjectID        string
	ParentMessageID  string
	Category         string
	ReadersAddresses string
	Body             []byte
	FilePath         string
	FileName         string
	FileType         string
	FileModifiedAt   time.Time
	Size             int64
	Checksum         string

	// Envelope fields, used for verification upon opening
	AuthorAddress string
	Readers       string
	ID            string
}

func NewMessage(authorEmailAddress string) (*Message, error) {
	authorEmailAddress, domain, localPart := address.ParseEmailAddress(authorEmailAddress)

	author, err := user.LocalUser(authorEmailAddress)
	if err != nil {
		return nil, err
	}

	messageId, err := utils.NewMessageID(domain, localPart)
	if err != nil {
		return nil, err
	}

	authorAsReader := user.AsReader(author)
	message := Message{
		ID:            *messageId,
		Author:        *author,
		IsBroadcast:   true,
		PayloadCipher: nil,
		Readers:       []user.Reader{*authorAsReader},
	}

	return &message, nil
}

func (msg *Message) IsFile() bool {
	return msg.Content.Category == "file" && msg.FileName != ""
}

func (msg *Message) AddReader(readerEmailAddress string) error {
	msg.IsBroadcast = false
	if msg.PayloadCipher == nil {
		msg.PayloadCipher = &crypto.CipherInfo{
			Algorithm: crypto.SYMMETRIC_CIPHER,
			Stream:    msg.IsFile(),
			ChunkSize: crypto.DEFAULT_CHUNK_SIZE,
		}
	}

	if msg.AccessKey == nil {
		password, err := crypto.GenerateRandomBytes(READER_KEY_LENGTH)
		if err != nil {
			return err
		}
		msg.AccessKey = password
	}

	//TODO: The reader comes from a remote profile, not local keys.
	// For development purposes, we however use local keys.

	u, err := user.LocalUser(readerEmailAddress)
	if err != nil {
		return err
	}
	r := user.AsReader(u)
	r.Link = linksPkg.Make(msg.Author.Address, r.Address)

	readers := append(msg.Readers, *r)
	msg.Readers = readers
	return nil
}

func ValidMessageSubject(subject string) bool {
	return len(subject) < MESSAGE_SUBJECT_MAXIMUM_LENGTH
}

func (msg *Message) SubjectRequired() bool {
	return (strings.TrimSpace(msg.Content.Subject) == "") &&
		((strings.TrimSpace(msg.Content.SubjectID) == "") || (msg.Content.SubjectID == msg.ID))
}

// Minimum stream identifier length is 1 characters=
// Maximum stream identifier length is 128 characters
// No URL unsafe characters, only alphanumerics
func ValidStreamID(streamID string) bool {
	if streamID == "" || len(streamID) > MESSAGE_STREAM_MAXIMUM_LENGTH {
		return false
	}
	return utils.StringIsAlphaNumeric(streamID)
}

func (msg *Message) SetStreamID(streamID string) bool {
	streamID = strings.TrimSpace(streamID)
	if !ValidStreamID(streamID) {
		return false
	}
	msg.StreamID = streamID
	return true
}

func (msg *Message) SetSubject(subject string) {
	subject = strings.TrimSpace(subject)
	if subject != "" {
		msg.Content.Subject = subject
	}
}

func (msg *Message) SetSubjectID(subjectID string) {
	subjectID = strings.ToLower(strings.TrimSpace(subjectID))
	if subjectID != "" {
		msg.Content.SubjectID = subjectID
	}
}

func (msg *Message) SetParentMessageID(messageID string) {
	msg.Content.ParentMessageID = messageID
}

func (msg *Message) SetCategory(category string) {
	msg.Content.Category = category
}

func (msg *Message) SetPlainContent(body []byte) {
	msg.Content.Body = body
	msg.Content.Size = int64(len(body))
	msg.Content.Checksum, _ = crypto.Checksum(body)
	msg.Content.Date = utils.TimestampNow()
	msg.Content.SubjectID = msg.ID
	msg.Content.ParentMessageID = msg.ID
	if msg.Content.Category == "" {
		msg.Content.Category = DEFAULT_MESSAGE_CATEGORY
	}

	// In case the message is encrypted
	msg.PayloadCipher.Stream = false
	msg.PayloadCipher.Algorithm = crypto.SYMMETRIC_CIPHER
}

// Note: Checksum is lazily calculated upon message encryption or copy,
// as otherwise we would be reading the file unnecessarily twice.
func (msg *Message) SetFileContent(filePath string) error {
	fileInfo, err := utils.GetFileInfo(filePath)
	if err != nil {
		return err
	}
	msg.Content.FilePath = fileInfo.Path
	msg.Content.FileName = fileInfo.BaseName
	msg.Content.FileType = fileInfo.Type
	msg.Content.FileModifiedAt = fileInfo.ModifiedAt
	msg.Content.Size = fileInfo.Size
	msg.Content.Date = utils.TimestampNow()
	msg.Content.SubjectID = msg.ID
	msg.Content.ParentMessageID = msg.ID
	// The category MUST be set to file when a file is provided
	msg.Content.Category = MESSAGE_FILE_CATEGORY

	// In case the message is encrypted
	msg.PayloadCipher.Stream = true
	msg.PayloadCipher.Algorithm = crypto.SYMMETRIC_FILE_CIPHER
	msg.PayloadCipher.ChunkSize = crypto.DEFAULT_CHUNK_SIZE

	return nil
}
