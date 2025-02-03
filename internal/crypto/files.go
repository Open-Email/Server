package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"email.mercata.com/internal/crypto/secretstream"
	"encoding/hex"
	"io"
	"os"
)

type IOInfo struct {
	InputSize      int64
	InputChecksum  string
	OutputSize     int64
	OutputChecksum string
}

const DEFAULT_CHUNK_SIZE = 8192
const MAX_CHUNK_SIZE = 1048576

func CopyFile(sourceFilePath, destFilePath string) (*IOInfo, error) {
	var inputSize int64
	plainHash := sha256.New()
	sourceFile, err := os.Open(sourceFilePath)
	if err != nil {
		return nil, err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(destFilePath)
	if err != nil {
		return nil, err
	}
	defer destFile.Close()

	buffer := make([]byte, DEFAULT_CHUNK_SIZE)

	for {
		n, err := sourceFile.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n > 0 {
			inputSize += int64(n)
			plainHash.Write(buffer[:n])
			if _, err := destFile.Write(buffer[:n]); err != nil {
				return nil, err
			}
		}
		if err == io.EOF {
			break
		}
	}
	plainChecksum := hex.EncodeToString(plainHash.Sum(nil))
	return &IOInfo{
		InputSize:      inputSize,
		InputChecksum:  plainChecksum,
		OutputSize:     inputSize,
		OutputChecksum: plainChecksum,
	}, nil
}

func SecretStreamXchacha20Poly1305EncryptFile(sourceFilePath, encryptedDestinationFilePath string, key []byte, chunkSize int64) (*IOInfo, error) {
	sourceFile, err := os.Open(sourceFilePath)
	if err != nil {
		return nil, err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(encryptedDestinationFilePath)
	if err != nil {
		return nil, err
	}
	defer destFile.Close()

	header := make([]byte, secretstream.HeaderBytes)
	if _, err := io.ReadFull(rand.Reader, header); err != nil {
		return nil, err
	}

	encryptor, err := secretstream.NewEncryptor(header, key)
	if err != nil {
		return nil, err
	}

	var inputSize int64
	var outputSize int64 = int64(secretstream.HeaderBytes)

	destFile.Write(header)

	plainHash := sha256.New()
	cipherHash := sha256.New()

	cipherHash.Write(header)

	buffer := make([]byte, chunkSize)

	eofReached := false
	var encryptorTag byte = secretstream.TagMessage

	for {
		n, err := sourceFile.Read(buffer)
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			eofReached = true
		}
		if n == 0 {
			break
		}
		if int64(n) < chunkSize {
			eofReached = true
		}
		plainHash.Write(buffer[:n])

		var encryptedBuffer []byte
		if eofReached {
			encryptorTag = secretstream.TagFinal
		}

		encryptedBuffer, err = encryptor.Push(buffer[:n], nil, encryptorTag)
		if err != nil {
			return nil, err
		}

		writtenBytes, err := destFile.Write(encryptedBuffer)
		if err != nil {
			return nil, err
		}
		inputSize += int64(n)
		outputSize += int64(writtenBytes)
		cipherHash.Write(encryptedBuffer)
	}

	cipherChecksum := cipherHash.Sum(nil)
	plainChecksum := plainHash.Sum(nil)
	return &IOInfo{
		InputSize:      inputSize,
		InputChecksum:  hex.EncodeToString(plainChecksum),
		OutputSize:     outputSize,
		OutputChecksum: hex.EncodeToString(cipherChecksum),
	}, nil
}

func SecretStreamXchacha20Poly1305DecryptFile(encryptedFilePath, decryptedFilePath string, key []byte, chunkSize int64) (*IOInfo, error) {
	encryptedFile, err := os.Open(encryptedFilePath)
	if err != nil {
		return nil, err
	}
	defer encryptedFile.Close()

	decryptedFile, err := os.Create(decryptedFilePath)
	if err != nil {
		return nil, err
	}
	defer decryptedFile.Close()

	plainHash := sha256.New()
	cipherHash := sha256.New()

	header := make([]byte, secretstream.HeaderBytes)
	_, err = encryptedFile.Read(header)
	if err != nil {
		return nil, err
	}

	decryptor, err := secretstream.NewDecryptor(header, key)
	if err != nil {
		return nil, err
	}

	cipherHash.Write(header)

	var inputSize int64 = int64(secretstream.HeaderBytes)
	var outputSize int64

	buffer := make([]byte, chunkSize+secretstream.AdditionalBytes)

	for {
		n, err := encryptedFile.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n == 0 {
			break
		}

		cipherHash.Write(buffer[:n])
		inputSize += int64(n)

		decryptedBuffer, tag, err := decryptor.Pull(buffer[:n], nil)
		if err != nil {
			return nil, err
		}

		if tag == secretstream.TagMessage || tag == secretstream.TagFinal {
			writtenBytes, err := decryptedFile.Write(decryptedBuffer)
			if err != nil {
				return nil, err
			}
			plainHash.Write(decryptedBuffer)
			outputSize += int64(writtenBytes)

			if tag == secretstream.TagFinal {
				break
			}
		}

	}
	encryptedFile.Close()

	cipherChecksum := cipherHash.Sum(nil)
	plainChecksum := plainHash.Sum(nil)
	return &IOInfo{
		InputSize:      inputSize,
		InputChecksum:  hex.EncodeToString(cipherChecksum),
		OutputSize:     outputSize,
		OutputChecksum: hex.EncodeToString(plainChecksum),
	}, nil
}
