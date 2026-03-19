package security

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestVerifyStartupIntegrityFromEnvMatchesExpectedHash(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "daemon-bin")
	if err := os.WriteFile(bin, []byte("hello-binary"), 0o700); err != nil {
		t.Fatalf("write test binary: %v", err)
	}
	hash, err := fileSHA256Hex(bin)
	if err != nil {
		t.Fatalf("hash test binary: %v", err)
	}

	prevExec := osExecutable
	osExecutable = func() (string, error) { return bin, nil }
	t.Cleanup(func() { osExecutable = prevExec })

	t.Setenv("KNIT_REQUIRE_BINARY_INTEGRITY", "1")
	t.Setenv("KNIT_BINARY_SHA256", hash)
	if err := VerifyStartupIntegrityFromEnv(); err != nil {
		t.Fatalf("expected integrity verification to pass, got %v", err)
	}
}

func TestVerifyStartupIntegrityFromEnvRejectsMismatch(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "daemon-bin")
	if err := os.WriteFile(bin, []byte("hello-binary"), 0o700); err != nil {
		t.Fatalf("write test binary: %v", err)
	}

	prevExec := osExecutable
	osExecutable = func() (string, error) { return bin, nil }
	t.Cleanup(func() { osExecutable = prevExec })

	t.Setenv("KNIT_REQUIRE_BINARY_INTEGRITY", "1")
	t.Setenv("KNIT_BINARY_SHA256", strings.Repeat("a", 64))
	if err := VerifyStartupIntegrityFromEnv(); err == nil {
		t.Fatalf("expected integrity mismatch error")
	}
}

func TestVerifyStartupIntegrityFromEnvUsesChecksumFile(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "daemon-bin")
	if err := os.WriteFile(bin, []byte("hello-binary"), 0o700); err != nil {
		t.Fatalf("write test binary: %v", err)
	}
	hash, err := fileSHA256Hex(bin)
	if err != nil {
		t.Fatalf("hash test binary: %v", err)
	}
	checksumFile := filepath.Join(dir, "checksums.txt")
	line := hash + "  " + filepath.Base(bin) + "\n"
	if err := os.WriteFile(checksumFile, []byte(line), 0o600); err != nil {
		t.Fatalf("write checksum file: %v", err)
	}

	prevExec := osExecutable
	osExecutable = func() (string, error) { return bin, nil }
	t.Cleanup(func() { osExecutable = prevExec })

	t.Setenv("KNIT_REQUIRE_BINARY_INTEGRITY", "1")
	t.Setenv("KNIT_BINARY_CHECKSUMS_FILE", checksumFile)
	if err := VerifyStartupIntegrityFromEnv(); err != nil {
		t.Fatalf("expected checksum file integrity verification to pass, got %v", err)
	}
}

func TestVerifyStartupIntegrityFromEnvRunsVerificationCommand(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "daemon-bin")
	if err := os.WriteFile(bin, []byte("hello-binary"), 0o700); err != nil {
		t.Fatalf("write test binary: %v", err)
	}

	prevExec := osExecutable
	osExecutable = func() (string, error) { return bin, nil }
	t.Cleanup(func() { osExecutable = prevExec })

	t.Setenv("KNIT_REQUIRE_BINARY_INTEGRITY", "1")
	verifyCmd := "echo verified {binary} {sha256} >/dev/null"
	if runtime.GOOS == "windows" {
		verifyCmd = "echo verified {binary} {sha256} > NUL"
	}
	t.Setenv("KNIT_VERIFY_SIGNATURE_COMMAND", verifyCmd)
	if err := VerifyStartupIntegrityFromEnv(); err != nil {
		t.Fatalf("expected verification command to pass, got %v", err)
	}
}

func TestVerifyStartupIntegrityFromEnvFailsVerificationCommand(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "daemon-bin")
	if err := os.WriteFile(bin, []byte("hello-binary"), 0o700); err != nil {
		t.Fatalf("write test binary: %v", err)
	}

	prevExec := osExecutable
	osExecutable = func() (string, error) { return bin, nil }
	t.Cleanup(func() { osExecutable = prevExec })

	t.Setenv("KNIT_REQUIRE_BINARY_INTEGRITY", "1")
	t.Setenv("KNIT_VERIFY_SIGNATURE_COMMAND", "echo bad && exit 42")
	if err := VerifyStartupIntegrityFromEnv(); err == nil {
		t.Fatalf("expected verification command failure")
	}
}

func TestVerifyStartupIntegrityFromEnvUsesSignedReleaseManifest(t *testing.T) {
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not available")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "daemon-bin")
	if err := os.WriteFile(bin, []byte("hello-binary"), 0o700); err != nil {
		t.Fatalf("write test binary: %v", err)
	}
	hash, err := fileSHA256Hex(bin)
	if err != nil {
		t.Fatalf("hash test binary: %v", err)
	}
	manifestPath := filepath.Join(dir, "release-manifest.json")
	manifestBody := `{"version":"test","checksums_file":"checksums.txt","signed_checksums":true,"artifacts":[{"sha256":"` + hash + `","path":"bin/linux_amd64/daemon-bin"}]}`
	if err := os.WriteFile(manifestPath, []byte(manifestBody), 0o600); err != nil {
		t.Fatalf("write release manifest: %v", err)
	}
	privateKey := filepath.Join(dir, "release-private.pem")
	publicKey := filepath.Join(dir, "release-public.pem")
	genKey := exec.Command("openssl", "genpkey", "-algorithm", "RSA", "-out", privateKey, "-pkeyopt", "rsa_keygen_bits:2048")
	if out, err := genKey.CombinedOutput(); err != nil {
		t.Fatalf("generate private key failed: %v\n%s", err, string(out))
	}
	genPub := exec.Command("openssl", "rsa", "-in", privateKey, "-pubout", "-out", publicKey)
	if out, err := genPub.CombinedOutput(); err != nil {
		t.Fatalf("generate public key failed: %v\n%s", err, string(out))
	}
	manifestSig := filepath.Join(dir, "release-manifest.sig")
	sign := exec.Command("openssl", "dgst", "-sha256", "-sign", privateKey, "-out", manifestSig, manifestPath)
	if out, err := sign.CombinedOutput(); err != nil {
		t.Fatalf("sign release manifest failed: %v\n%s", err, string(out))
	}

	prevExec := osExecutable
	osExecutable = func() (string, error) { return bin, nil }
	t.Cleanup(func() { osExecutable = prevExec })

	t.Setenv("KNIT_REQUIRE_BINARY_INTEGRITY", "1")
	t.Setenv("KNIT_RELEASE_MANIFEST_FILE", manifestPath)
	t.Setenv("KNIT_RELEASE_MANIFEST_SIGNATURE_FILE", manifestSig)
	t.Setenv("RELEASE_SIGNING_PUBLIC_KEY", publicKey)
	if err := VerifyStartupIntegrityFromEnv(); err != nil {
		t.Fatalf("expected signed release manifest verification to pass, got %v", err)
	}
}

func TestVerifyStartupIntegrityFromEnvRejectsUnsignedReleaseManifestMismatch(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "daemon-bin")
	if err := os.WriteFile(bin, []byte("hello-binary"), 0o700); err != nil {
		t.Fatalf("write test binary: %v", err)
	}
	manifestPath := filepath.Join(dir, "release-manifest.json")
	manifestBody := `{"version":"test","checksums_file":"checksums.txt","signed_checksums":true,"artifacts":[{"sha256":"` + strings.Repeat("a", 64) + `","path":"bin/linux_amd64/daemon-bin"}]}`
	if err := os.WriteFile(manifestPath, []byte(manifestBody), 0o600); err != nil {
		t.Fatalf("write release manifest: %v", err)
	}

	prevExec := osExecutable
	osExecutable = func() (string, error) { return bin, nil }
	t.Cleanup(func() { osExecutable = prevExec })

	t.Setenv("KNIT_REQUIRE_BINARY_INTEGRITY", "1")
	t.Setenv("KNIT_RELEASE_MANIFEST_FILE", manifestPath)
	if err := VerifyStartupIntegrityFromEnv(); err == nil {
		t.Fatalf("expected release manifest mismatch to fail integrity verification")
	}
}
