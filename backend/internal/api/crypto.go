package api

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// fieldEncryptionPrefix marks a value as already encrypted in the database.
const fieldEncryptionPrefix = "enc:"

var (
	fieldCipher     cipher.AEAD
	fieldCipherOnce sync.Once
	fieldCipherErr  error
)

// getFieldCipher lazily initializes and returns the AES-256-GCM cipher
// used for encrypting sensitive fields like IPMI passwords.
// The key is read from the FIELD_ENCRYPTION_KEY environment variable (hex-encoded, 32 bytes).
// If the env var is not set, encryption is disabled and values are stored in plaintext.
func getFieldCipher() (cipher.AEAD, error) {
	fieldCipherOnce.Do(func() {
		keyHex := os.Getenv("FIELD_ENCRYPTION_KEY")
		if keyHex == "" {
			fieldCipherErr = errors.New("FIELD_ENCRYPTION_KEY not set, field encryption disabled")
			return
		}

		key, err := decodeHexKey(keyHex)
		if err != nil {
			fieldCipherErr = fmt.Errorf("invalid FIELD_ENCRYPTION_KEY: %w", err)
			return
		}

		block, err := aes.NewCipher(key)
		if err != nil {
			fieldCipherErr = fmt.Errorf("failed to create AES cipher: %w", err)
			return
		}

		fieldCipher, fieldCipherErr = cipher.NewGCM(block)
	})
	return fieldCipher, fieldCipherErr
}

// EncryptField encrypts a plaintext string using AES-256-GCM.
// Returns "enc:<base64-ciphertext>" on success, or the original value if encryption is unavailable.
func EncryptField(plaintext string) string {
	if plaintext == "" || strings.HasPrefix(plaintext, fieldEncryptionPrefix) {
		return plaintext // already encrypted or empty
	}

	gcm, err := getFieldCipher()
	if err != nil {
		return plaintext // encryption not available, store as-is
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return plaintext
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return fieldEncryptionPrefix + base64.StdEncoding.EncodeToString(ciphertext)
}

// DecryptField decrypts an "enc:<base64>" value back to plaintext.
// Returns the original string if it's not encrypted or decryption fails.
func DecryptField(value string) string {
	if !strings.HasPrefix(value, fieldEncryptionPrefix) {
		return value // not encrypted
	}

	gcm, err := getFieldCipher()
	if err != nil {
		return value // can't decrypt without key
	}

	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, fieldEncryptionPrefix))
	if err != nil {
		return value
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return value
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return value
	}

	return string(plaintext)
}

// decodeHexKey parses a hex string into a 32-byte AES key.
func decodeHexKey(hex string) ([]byte, error) {
	hex = strings.TrimSpace(hex)
	if len(hex) != 64 {
		return nil, fmt.Errorf("key must be 64 hex characters (32 bytes), got %d", len(hex))
	}

	key := make([]byte, 32)
	for i := 0; i < 32; i++ {
		b, err := parseHexByte(hex[i*2 : i*2+2])
		if err != nil {
			return nil, err
		}
		key[i] = b
	}
	return key, nil
}

func parseHexByte(s string) (byte, error) {
	var b byte
	for _, c := range s {
		b <<= 4
		switch {
		case c >= '0' && c <= '9':
			b |= byte(c - '0')
		case c >= 'a' && c <= 'f':
			b |= byte(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			b |= byte(c - 'A' + 10)
		default:
			return 0, fmt.Errorf("invalid hex character: %c", c)
		}
	}
	return b, nil
}
