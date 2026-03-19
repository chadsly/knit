package security

import "testing"

func TestZeroBytesWipesBuffer(t *testing.T) {
	buf := []byte("super-secret")
	ZeroBytes(buf)
	for i, b := range buf {
		if b != 0 {
			t.Fatalf("expected byte %d to be zeroed, got %d", i, b)
		}
	}
}

func TestNewEncryptorCopiesKeyIndependently(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	enc, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("new encryptor: %v", err)
	}
	ZeroBytes(key)
	ciphertext, err := enc.Encrypt([]byte("hello"))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	plain, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(plain) != "hello" {
		t.Fatalf("unexpected decrypt result: %q", string(plain))
	}
}
