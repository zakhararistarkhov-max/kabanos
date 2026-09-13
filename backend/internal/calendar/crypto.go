package calendar

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
)

// DeriveKey builds a 32-byte AES key. If a dedicated key is configured it is
// used (hashed to a fixed length); otherwise it is derived from the JWT secret
// so the feature works out of the box. Rotating the source invalidates stored
// passwords — users simply reconnect.
func DeriveKey(configuredKey, jwtSecret []byte) []byte {
	src := configuredKey
	if len(src) == 0 {
		src = append([]byte("kabanos-calendar-v1:"), jwtSecret...)
	}
	sum := sha256.Sum256(src)
	return sum[:]
}

type cryptor struct{ aead cipher.AEAD }

func newCryptor(key []byte) (cryptor, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return cryptor{}, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return cryptor{}, err
	}
	return cryptor{aead: aead}, nil
}

// encrypt returns nonce || ciphertext.
func (c cryptor) encrypt(plaintext string) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return c.aead.Seal(nonce, nonce, []byte(plaintext), nil), nil
}

func (c cryptor) decrypt(blob []byte) (string, error) {
	ns := c.aead.NonceSize()
	if len(blob) < ns {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := blob[:ns], blob[ns:]
	pt, err := c.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}
