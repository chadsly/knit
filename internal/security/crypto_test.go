package security

import "testing"

func TestEncryptorRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("new encryptor: %v", err)
	}
	cipher, err := enc.Encrypt([]byte("hello"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	plain, err := enc.Decrypt(cipher)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(plain) != "hello" {
		t.Fatalf("unexpected decrypt result: %s", string(plain))
	}
}
