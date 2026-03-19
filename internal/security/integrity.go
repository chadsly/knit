package security

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var osExecutable = os.Executable

func VerifyStartupIntegrityFromEnv() error {
	require := parseEnvBool(os.Getenv("KNIT_REQUIRE_BINARY_INTEGRITY"))
	expectedHash := strings.TrimSpace(strings.ToLower(os.Getenv("KNIT_BINARY_SHA256")))
	checksumFile := strings.TrimSpace(os.Getenv("KNIT_BINARY_CHECKSUMS_FILE"))
	verifyCmd := strings.TrimSpace(os.Getenv("KNIT_VERIFY_SIGNATURE_COMMAND"))
	releaseManifestFile := strings.TrimSpace(os.Getenv("KNIT_RELEASE_MANIFEST_FILE"))
	releaseManifestSigFile := strings.TrimSpace(os.Getenv("KNIT_RELEASE_MANIFEST_SIGNATURE_FILE"))
	releasePublicKey := strings.TrimSpace(os.Getenv("RELEASE_SIGNING_PUBLIC_KEY"))

	if !require && expectedHash == "" && checksumFile == "" && verifyCmd == "" && releaseManifestFile == "" && releaseManifestSigFile == "" {
		return nil
	}
	exePath, err := osExecutable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("resolve executable absolute path: %w", err)
	}
	actualHash, err := fileSHA256Hex(exePath)
	if err != nil {
		return fmt.Errorf("hash executable: %w", err)
	}

	if expectedHash != "" && actualHash != expectedHash {
		return fmt.Errorf("binary integrity mismatch for %s", filepath.Base(exePath))
	}
	if checksumFile != "" {
		ok, lookupErr := verifyHashInChecksumFile(checksumFile, exePath, actualHash)
		if lookupErr != nil {
			return lookupErr
		}
		if !ok {
			return fmt.Errorf("checksum entry not found for executable in %s", checksumFile)
		}
	}
	if releaseManifestFile != "" || releaseManifestSigFile != "" {
		if err := verifyReleaseManifest(releaseManifestFile, releaseManifestSigFile, releasePublicKey, exePath, actualHash); err != nil {
			return err
		}
	}
	if verifyCmd != "" {
		cmdText := strings.ReplaceAll(verifyCmd, "{binary}", shellEscape(exePath))
		cmdText = strings.ReplaceAll(cmdText, "{sha256}", actualHash)
		cmd := shellCommandForIntegrity(cmdText)
		if out, runErr := cmd.CombinedOutput(); runErr != nil {
			msg := strings.TrimSpace(string(out))
			if msg == "" {
				msg = runErr.Error()
			}
			return fmt.Errorf("signature verification command failed: %s", msg)
		}
	}
	return nil
}

type releaseManifest struct {
	Version         string `json:"version"`
	ChecksumsFile   string `json:"checksums_file"`
	SignedChecksums bool   `json:"signed_checksums"`
	Artifacts       []struct {
		SHA256 string `json:"sha256"`
		Path   string `json:"path"`
	} `json:"artifacts"`
}

func fileSHA256Hex(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func verifyHashInChecksumFile(path, binaryPath, expectedHash string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("open checksum file: %w", err)
	}
	defer f.Close()

	base := filepath.Base(binaryPath)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		hash := strings.ToLower(strings.TrimSpace(fields[0]))
		target := strings.TrimSpace(fields[len(fields)-1])
		target = strings.TrimPrefix(target, "*")
		target = filepath.Base(target)
		if target == base && hash == expectedHash {
			return true, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("read checksum file: %w", err)
	}
	return false, nil
}

func verifyReleaseManifest(manifestPath, manifestSigPath, publicKeyPath, binaryPath, expectedHash string) error {
	if strings.TrimSpace(manifestPath) == "" {
		return fmt.Errorf("release manifest file is required when manifest verification is enabled")
	}
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read release manifest: %w", err)
	}
	if strings.TrimSpace(manifestSigPath) != "" {
		if strings.TrimSpace(publicKeyPath) == "" {
			return fmt.Errorf("release manifest signature provided but RELEASE_SIGNING_PUBLIC_KEY is not set")
		}
		cmd := exec.Command("openssl", "dgst", "-sha256", "-verify", publicKeyPath, "-signature", manifestSigPath, manifestPath)
		if out, runErr := cmd.CombinedOutput(); runErr != nil {
			msg := strings.TrimSpace(string(out))
			if msg == "" {
				msg = runErr.Error()
			}
			return fmt.Errorf("release manifest signature verification failed: %s", msg)
		}
	}
	var manifest releaseManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return fmt.Errorf("decode release manifest: %w", err)
	}
	base := filepath.Base(binaryPath)
	for _, artifact := range manifest.Artifacts {
		if strings.EqualFold(filepath.Base(strings.TrimSpace(artifact.Path)), base) && strings.EqualFold(strings.TrimSpace(artifact.SHA256), expectedHash) {
			return nil
		}
	}
	return fmt.Errorf("binary %s hash not present in signed release manifest", base)
}

func shellCommandForIntegrity(raw string) *exec.Cmd {
	if strings.TrimSpace(raw) == "" {
		return exec.Command("true")
	}
	if runtimeGOOS() == "windows" {
		return exec.Command("cmd", "/C", raw)
	}
	return exec.Command("sh", "-lc", raw)
}

var runtimeGOOS = func() string {
	return runtime.GOOS
}

func parseEnvBool(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func shellEscape(v string) string {
	if v == "" {
		return "''"
	}
	v = strings.ReplaceAll(v, `'`, `'\''`)
	return "'" + v + "'"
}
