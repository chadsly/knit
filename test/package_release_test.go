package test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPackageReleaseScriptProducesArchivesAndChecksums(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}

	distDir := prepareFakeDist(t)
	writeReleaseInputs(t, distDir)
	cmd := exec.Command("bash", "../scripts/package-release.sh", distDir)
	cmd.Env = append(os.Environ(), "VERSION=0.0.0-test")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("package script failed: %v\n%s", err, string(out))
	}

	pkgDir := filepath.Join(distDir, "packages")
	checksumsPath := filepath.Join(pkgDir, "checksums.txt")
	if _, err := os.Stat(checksumsPath); err != nil {
		t.Fatalf("expected checksums file: %v", err)
	}
	checksums, err := os.ReadFile(checksumsPath)
	if err != nil {
		t.Fatalf("read checksums: %v", err)
	}
	if !strings.Contains(string(checksums), ".tar.gz") || !strings.Contains(string(checksums), ".zip") {
		t.Fatalf("expected both tar.gz and zip checksums, got:\n%s", string(checksums))
	}
	if !strings.Contains(string(checksums), ".install.sh") || !strings.Contains(string(checksums), ".install.ps1") {
		t.Fatalf("expected installer helper checksums, got:\n%s", string(checksums))
	}
	if _, err := os.Stat(filepath.Join(pkgDir, "installers", "knit_0.0.0-test_linux_amd64.install.sh")); err != nil {
		t.Fatalf("expected linux installer helper: %v", err)
	}
	if _, err := os.Stat(filepath.Join(pkgDir, "installers", "knit_0.0.0-test_windows_amd64.install.ps1")); err != nil {
		t.Fatalf("expected windows installer helper: %v", err)
	}
	if _, err := os.Stat(filepath.Join(pkgDir, "release-manifest.json")); err != nil {
		t.Fatalf("expected release manifest: %v", err)
	}
	if _, err := os.Stat(filepath.Join(pkgDir, "build-manifest.json")); err != nil {
		t.Fatalf("expected copied build manifest: %v", err)
	}
	if _, err := os.Stat(filepath.Join(pkgDir, "sbom.spdx.json")); err != nil {
		t.Fatalf("expected copied sbom: %v", err)
	}
	if _, err := os.Stat(filepath.Join(pkgDir, "dependency-scan.json")); err != nil {
		t.Fatalf("expected copied dependency scan report: %v", err)
	}
	if _, err := os.Stat(filepath.Join(pkgDir, "npm", "knit-daemon", "package.json")); err != nil {
		t.Fatalf("expected npm package scaffold: %v", err)
	}
}

func TestPackageReleaseScriptBuildsSelfContainedNpmPackage(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not available")
	}

	distDir := prepareFakeDist(t)
	writeReleaseInputs(t, distDir)
	cmd := exec.Command("bash", "../scripts/package-release.sh", distDir)
	cmd.Env = append(os.Environ(), "VERSION=0.0.0-test")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("package script failed: %v\n%s", err, string(out))
	}

	npmPkgDir := filepath.Join(distDir, "packages", "npm", "knit-daemon")
	install := exec.Command("node", "./lib/install.js")
	install.Dir = npmPkgDir
	if out, err := install.CombinedOutput(); err != nil {
		t.Fatalf("npm installer failed: %v\n%s", err, string(out))
	}
	runtimeDir := filepath.Join(npmPkgDir, "artifacts", "runtime")
	if _, err := os.Stat(filepath.Join(runtimeDir, "daemon")); err != nil && runtime.GOOS != "windows" {
		t.Fatalf("expected extracted daemon: %v", err)
	}
	if _, err := os.Stat(filepath.Join(npmPkgDir, "artifacts", "release-manifest.json")); err != nil {
		t.Fatalf("expected bundled release manifest: %v", err)
	}
}

func TestPackageReleaseScriptSupportsChecksumSigning(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not available")
	}

	distDir := prepareFakeDist(t)
	writeReleaseInputs(t, distDir)
	keysDir := t.TempDir()
	privateKey := filepath.Join(keysDir, "release-private.pem")
	publicKey := filepath.Join(keysDir, "release-public.pem")

	genKey := exec.Command("openssl", "genpkey", "-algorithm", "RSA", "-out", privateKey, "-pkeyopt", "rsa_keygen_bits:2048")
	if out, err := genKey.CombinedOutput(); err != nil {
		t.Fatalf("generate private key failed: %v\n%s", err, string(out))
	}
	genPub := exec.Command("openssl", "rsa", "-in", privateKey, "-pubout", "-out", publicKey)
	if out, err := genPub.CombinedOutput(); err != nil {
		t.Fatalf("generate public key failed: %v\n%s", err, string(out))
	}

	cmd := exec.Command("bash", "../scripts/package-release.sh", distDir)
	cmd.Env = append(os.Environ(),
		"VERSION=0.0.1-test",
		"RELEASE_SIGNING_PRIVATE_KEY="+privateKey,
		"RELEASE_SIGNING_PUBLIC_KEY="+publicKey,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("package signing failed: %v\n%s", err, string(out))
	}
	sigPath := filepath.Join(distDir, "packages", "checksums.sig")
	if _, err := os.Stat(sigPath); err != nil {
		t.Fatalf("expected checksums signature file: %v", err)
	}
	manifestSigPath := filepath.Join(distDir, "packages", "release-manifest.sig")
	if _, err := os.Stat(manifestSigPath); err != nil {
		t.Fatalf("expected release manifest signature file: %v", err)
	}

	verify := exec.Command("bash", "../scripts/verify-update-signature.sh", publicKey, filepath.Join(distDir, "packages", "checksums.txt"), sigPath, filepath.Join(distDir, "packages", "knit_0.0.1-test_windows_amd64.zip"))
	if out, err := verify.CombinedOutput(); err != nil {
		t.Fatalf("verify update signature failed: %v\n%s", err, string(out))
	}
}

func TestReleaseReadinessCheckPassesForUnsignedNonTagBuild(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	distDir := prepareFakeDist(t)
	writeReleaseInputs(t, distDir)
	cmd := exec.Command("bash", "../scripts/package-release.sh", distDir)
	cmd.Env = append(os.Environ(), "VERSION=0.0.2-test")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("package script failed: %v\n%s", err, string(out))
	}

	check := exec.Command("bash", "../scripts/release-readiness-check.sh", filepath.Join(distDir, "packages"))
	if out, err := check.CombinedOutput(); err != nil {
		t.Fatalf("release-readiness check failed: %v\n%s", err, string(out))
	}
}

func TestReleaseReadinessCheckFailsForTagWithoutSignature(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	distDir := prepareFakeDist(t)
	writeReleaseInputs(t, distDir)
	cmd := exec.Command("bash", "../scripts/package-release.sh", distDir)
	cmd.Env = append(os.Environ(), "VERSION=0.0.3-test")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("package script failed: %v\n%s", err, string(out))
	}

	check := exec.Command("bash", "../scripts/release-readiness-check.sh", filepath.Join(distDir, "packages"))
	check.Env = append(os.Environ(), "CI_COMMIT_TAG=v0.0.3")
	out, err := check.CombinedOutput()
	if err == nil {
		t.Fatalf("expected release-readiness check to fail for tag without signature")
	}
	if !strings.Contains(string(out), "requires signed checksums") {
		t.Fatalf("expected tagged signature error, got:\n%s", string(out))
	}
}

func TestReleaseReadinessCheckFailsForTagWithoutManifestSignature(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not available")
	}
	distDir := prepareFakeDist(t)
	writeReleaseInputs(t, distDir)
	keysDir := t.TempDir()
	privateKey := filepath.Join(keysDir, "release-private.pem")
	publicKey := filepath.Join(keysDir, "release-public.pem")
	genKey := exec.Command("openssl", "genpkey", "-algorithm", "RSA", "-out", privateKey, "-pkeyopt", "rsa_keygen_bits:2048")
	if out, err := genKey.CombinedOutput(); err != nil {
		t.Fatalf("generate private key failed: %v\n%s", err, string(out))
	}
	genPub := exec.Command("openssl", "rsa", "-in", privateKey, "-pubout", "-out", publicKey)
	if out, err := genPub.CombinedOutput(); err != nil {
		t.Fatalf("generate public key failed: %v\n%s", err, string(out))
	}
	cmd := exec.Command("bash", "../scripts/package-release.sh", distDir)
	cmd.Env = append(os.Environ(),
		"VERSION=0.0.31-test",
		"RELEASE_SIGNING_PRIVATE_KEY="+privateKey,
		"RELEASE_SIGNING_PUBLIC_KEY="+publicKey,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("package script failed: %v\n%s", err, string(out))
	}
	if err := os.Remove(filepath.Join(distDir, "packages", "release-manifest.sig")); err != nil {
		t.Fatalf("remove release manifest signature: %v", err)
	}
	check := exec.Command("bash", "../scripts/release-readiness-check.sh", filepath.Join(distDir, "packages"))
	check.Env = append(os.Environ(), "CI_COMMIT_TAG=v0.0.31", "RELEASE_SIGNING_PUBLIC_KEY="+publicKey)
	out, err := check.CombinedOutput()
	if err == nil {
		t.Fatalf("expected release-readiness check to fail for missing release manifest signature")
	}
	if !strings.Contains(string(out), "requires signed release manifest") {
		t.Fatalf("expected manifest signature error, got:\n%s", string(out))
	}
}

func TestReleaseReadinessCheckFailsForTaggedBuildWithoutSBOM(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not available")
	}
	distDir := prepareFakeDist(t)
	writeReleaseInputs(t, distDir)
	if err := os.Remove(filepath.Join(distDir, "sbom.spdx.json")); err != nil {
		t.Fatalf("remove sbom: %v", err)
	}
	keysDir := t.TempDir()
	privateKey := filepath.Join(keysDir, "release-private.pem")
	publicKey := filepath.Join(keysDir, "release-public.pem")
	genKey := exec.Command("openssl", "genpkey", "-algorithm", "RSA", "-out", privateKey, "-pkeyopt", "rsa_keygen_bits:2048")
	if out, err := genKey.CombinedOutput(); err != nil {
		t.Fatalf("generate private key failed: %v\n%s", err, string(out))
	}
	genPub := exec.Command("openssl", "rsa", "-in", privateKey, "-pubout", "-out", publicKey)
	if out, err := genPub.CombinedOutput(); err != nil {
		t.Fatalf("generate public key failed: %v\n%s", err, string(out))
	}
	cmd := exec.Command("bash", "../scripts/package-release.sh", distDir)
	cmd.Env = append(os.Environ(),
		"VERSION=0.0.4-test",
		"RELEASE_SIGNING_PRIVATE_KEY="+privateKey,
		"RELEASE_SIGNING_PUBLIC_KEY="+publicKey,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("package script failed: %v\n%s", err, string(out))
	}

	check := exec.Command("bash", "../scripts/release-readiness-check.sh", filepath.Join(distDir, "packages"))
	check.Env = append(os.Environ(), "CI_COMMIT_TAG=v0.0.4", "RELEASE_SIGNING_PUBLIC_KEY="+publicKey)
	out, err := check.CombinedOutput()
	if err == nil {
		t.Fatalf("expected release-readiness check to fail for missing sbom")
	}
	if !strings.Contains(string(out), "requires sbom file") {
		t.Fatalf("expected tagged sbom error, got:\n%s", string(out))
	}
}

func TestBuildCrossPlatformScriptProducesBuildManifest(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	distDir := t.TempDir()
	cmd := exec.Command("bash", "../scripts/build-cross-platform.sh", distDir)
	cmd.Env = append(os.Environ(), "VERSION=0.0.5-test", "SOURCE_DATE_EPOCH=123456789")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build script failed: %v\n%s", err, string(out))
	}
	manifestPath := filepath.Join(distDir, "build-manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read build manifest: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode build manifest: %v", err)
	}
	if payload["version"] != "0.0.5-test" {
		t.Fatalf("expected build manifest version, got %#v", payload["version"])
	}
	if payload["source_date_epoch"] != "123456789" {
		t.Fatalf("expected source date epoch, got %#v", payload["source_date_epoch"])
	}
	targets, _ := payload["targets"].([]any)
	expectedTargets := map[string]bool{
		"darwin/amd64":  false,
		"darwin/arm64":  false,
		"linux/amd64":   false,
		"linux/arm64":   false,
		"windows/amd64": false,
	}
	for _, raw := range targets {
		if target, ok := raw.(string); ok {
			if _, exists := expectedTargets[target]; exists {
				expectedTargets[target] = true
			}
		}
	}
	for target, found := range expectedTargets {
		if !found {
			t.Fatalf("expected build manifest target %s, got %#v", target, payload["targets"])
		}
	}
	artifacts, _ := payload["artifacts"].([]any)
	seenArtifacts := map[string]bool{
		"bin/darwin_amd64/daemon":      false,
		"bin/darwin_amd64/ui":          false,
		"bin/darwin_arm64/daemon":      false,
		"bin/darwin_arm64/ui":          false,
		"bin/linux_amd64/daemon":       false,
		"bin/linux_amd64/ui":           false,
		"bin/linux_arm64/daemon":       false,
		"bin/linux_arm64/ui":           false,
		"bin/windows_amd64/daemon.exe": false,
		"bin/windows_amd64/ui.exe":     false,
	}
	for _, raw := range artifacts {
		entry, _ := raw.(map[string]any)
		path, _ := entry["path"].(string)
		if _, exists := seenArtifacts[path]; exists {
			seenArtifacts[path] = true
		}
	}
	for path, found := range seenArtifacts {
		if !found {
			t.Fatalf("expected build manifest artifact %s, got %#v", path, payload["artifacts"])
		}
	}
}

func TestRuntimeSmokeScriptPassesForHostBuild(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	if _, err := exec.LookPath("curl"); err != nil {
		t.Skip("curl not available")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	distDir := t.TempDir()
	build := exec.Command("bash", "../scripts/build-cross-platform.sh", distDir)
	build.Env = append(os.Environ(), "VERSION=0.0.6-test")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build script failed: %v\n%s", err, string(out))
	}
	smoke := exec.Command("bash", "../scripts/runtime-smoke.sh", distDir)
	if out, err := smoke.CombinedOutput(); err != nil {
		t.Fatalf("runtime smoke failed: %v\n%s", err, string(out))
	}
}

func TestGenerateSBOMScriptProducesOutput(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	outFile := filepath.Join(t.TempDir(), "sbom.spdx.json")
	cmd := exec.Command("bash", "../scripts/generate-sbom.sh", outFile)
	cmd.Env = append(os.Environ(), "SOURCE_DATE_EPOCH=123456789")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generate sbom failed: %v\n%s", err, string(out))
	}
	raw, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read sbom: %v", err)
	}
	if !strings.Contains(string(raw), "SPDX-2.3") {
		t.Fatalf("expected SPDX output, got %s", string(raw))
	}
}

func TestDependencyScanScriptProducesOutput(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}
	outFile := filepath.Join(t.TempDir(), "dependency-scan.json")
	cmd := exec.Command("bash", "../scripts/dependency-scan.sh", outFile)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("dependency scan failed: %v\n%s", err, string(out))
	}
	raw, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read dependency scan output: %v", err)
	}
	if len(raw) == 0 {
		t.Fatalf("expected dependency scan output")
	}
}

func writeReleaseInputs(t *testing.T, distDir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(distDir, "build-manifest.json"), []byte(`{"version":"test"}`), 0o644); err != nil {
		t.Fatalf("write build manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distDir, "sbom.spdx.json"), []byte(`{"spdxVersion":"SPDX-2.3"}`), 0o644); err != nil {
		t.Fatalf("write sbom: %v", err)
	}
	if err := os.WriteFile(filepath.Join(distDir, "dependency-scan.json"), []byte(`{"status":"ok"}`), 0o644); err != nil {
		t.Fatalf("write dependency scan: %v", err)
	}
}

func prepareFakeDist(t *testing.T) string {
	t.Helper()
	distDir := t.TempDir()
	linuxDir := filepath.Join(distDir, "bin", "linux_amd64")
	linuxArmDir := filepath.Join(distDir, "bin", "linux_arm64")
	darwinDir := filepath.Join(distDir, "bin", "darwin_arm64")
	winDir := filepath.Join(distDir, "bin", "windows_amd64")
	if err := os.MkdirAll(linuxDir, 0o755); err != nil {
		t.Fatalf("mkdir linux bin dir: %v", err)
	}
	if err := os.MkdirAll(linuxArmDir, 0o755); err != nil {
		t.Fatalf("mkdir linux arm bin dir: %v", err)
	}
	if err := os.MkdirAll(darwinDir, 0o755); err != nil {
		t.Fatalf("mkdir darwin bin dir: %v", err)
	}
	if err := os.MkdirAll(winDir, 0o755); err != nil {
		t.Fatalf("mkdir windows bin dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(linuxDir, "daemon"), []byte("linux-daemon"), 0o755); err != nil {
		t.Fatalf("write linux daemon: %v", err)
	}
	if err := os.WriteFile(filepath.Join(linuxDir, "ui"), []byte("linux-ui"), 0o755); err != nil {
		t.Fatalf("write linux ui: %v", err)
	}
	if err := os.WriteFile(filepath.Join(linuxArmDir, "daemon"), []byte("linux-arm-daemon"), 0o755); err != nil {
		t.Fatalf("write linux arm daemon: %v", err)
	}
	if err := os.WriteFile(filepath.Join(linuxArmDir, "ui"), []byte("linux-arm-ui"), 0o755); err != nil {
		t.Fatalf("write linux arm ui: %v", err)
	}
	if err := os.WriteFile(filepath.Join(darwinDir, "daemon"), []byte("darwin-daemon"), 0o755); err != nil {
		t.Fatalf("write darwin daemon: %v", err)
	}
	if err := os.WriteFile(filepath.Join(darwinDir, "ui"), []byte("darwin-ui"), 0o755); err != nil {
		t.Fatalf("write darwin ui: %v", err)
	}
	if err := os.WriteFile(filepath.Join(winDir, "daemon.exe"), []byte("windows-daemon"), 0o755); err != nil {
		t.Fatalf("write windows daemon: %v", err)
	}
	if err := os.WriteFile(filepath.Join(winDir, "ui.exe"), []byte("windows-ui"), 0o755); err != nil {
		t.Fatalf("write windows ui: %v", err)
	}
	if runtime.GOOS == "windows" {
		t.Skip("packaging script tests assume GNU tar/zip behavior")
	}
	return distDir
}
