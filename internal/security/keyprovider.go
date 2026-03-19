package security

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	serviceName = "knit"
	keyUser     = "artifact-encryption-key"
)

// ResolveKey returns a 32-byte key from env override or OS keychain-backed secret storage.
func ResolveKey() ([]byte, error) {
	if v := strings.TrimSpace(os.Getenv("KNIT_ENCRYPTION_KEY_B64")); v != "" {
		decoded, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return nil, fmt.Errorf("decode KNIT_ENCRYPTION_KEY_B64: %w", err)
		}
		if len(decoded) != 32 {
			return nil, fmt.Errorf("KNIT_ENCRYPTION_KEY_B64 must decode to 32 bytes")
		}
		return decoded, nil
	}

	existing, err := keyring.Get(serviceName, keyUser)
	if err == nil && strings.TrimSpace(existing) != "" {
		decoded, decErr := base64.StdEncoding.DecodeString(existing)
		if decErr != nil {
			return nil, fmt.Errorf("decode key from OS secure storage: %w", decErr)
		}
		if len(decoded) != 32 {
			return nil, fmt.Errorf("stored key has invalid length")
		}
		return decoded, nil
	}

	generated := make([]byte, 32)
	if _, err := rand.Read(generated); err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(generated)
	if err := keyring.Set(serviceName, keyUser, encoded); err != nil {
		return nil, fmt.Errorf("store key in OS secure storage: %w", err)
	}
	return generated, nil
}
