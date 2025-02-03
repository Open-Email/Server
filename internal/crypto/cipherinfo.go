package crypto

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type CipherInfo struct {
	Algorithm string
	Stream    bool
	ChunkSize int64

	OriginalHeaderValue string
}

const ALGORITHM_ATTRIBUTE = "algorithm"
const CHUNK_SIZE_ATTRIBUTE = "chunk-size"

func (ci *CipherInfo) ToHeader() string {
	if ci.Stream {
		if ci.Algorithm == "" || ci.ChunkSize == 0 {
			return ""
		}
		return fmt.Sprintf("%s=%s; %s=%d", ALGORITHM_ATTRIBUTE, ci.Algorithm, CHUNK_SIZE_ATTRIBUTE, ci.ChunkSize)
	}
	return fmt.Sprintf("%s=%s", ALGORITHM_ATTRIBUTE, ci.Algorithm)
}

func CipherInfoFromHeader(headerValue string) (*CipherInfo, error) {
	ci := CipherInfo{
		Algorithm:           "",
		ChunkSize:           0,
		OriginalHeaderValue: headerValue,
	}

	parts := strings.Split(headerValue, ";")

	for _, pair := range parts {
		kvs := strings.Split(pair, "=")
		switch strings.ToLower(strings.TrimSpace(kvs[0])) {
		case CHUNK_SIZE_ATTRIBUTE:
			size, err := strconv.ParseInt(strings.TrimSpace(kvs[1]), 10, 64)
			if err != nil {
				return nil, err
			}
			if size > MAX_CHUNK_SIZE {
				return nil, errors.New("unacceptable chunk size")
			}
			ci.ChunkSize = size

		case ALGORITHM_ATTRIBUTE:
			algorithm := strings.ToLower(strings.TrimSpace(kvs[1]))
			if (algorithm == SYMMETRIC_CIPHER) || (algorithm == SYMMETRIC_FILE_CIPHER) {
				ci.Algorithm = algorithm
				ci.Stream = (algorithm == SYMMETRIC_FILE_CIPHER)
				continue
			}
			return nil, errors.New("unsupported encryption algorithm: " + algorithm)

		default:
			continue
		}
	}
	return &ci, nil
}
