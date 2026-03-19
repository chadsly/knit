package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

type Encryptor struct {
	key []byte
}

func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes")
	}
	out := make([]byte, 32)
	copy(out, key)
	return &Encryptor{key: out}, nil
}

func (e *Encryptor) Encrypt(plaintext []byte) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	defer ZeroBytes(nonce)
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	defer ZeroBytes(ciphertext)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (e *Encryptor) Decrypt(encoded string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	defer ZeroBytes(ciphertext)
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, payload := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plain, err := gcm.Open(nil, nonce, payload, nil)
	if err != nil {
		return nil, err
	}
	return plain, nil
}
