package test

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestSecurityAndBuildScriptsExist(t *testing.T) {
	files := []string{
		"../scripts/build-cross-platform.sh",
		"../scripts/package-release.sh",
		"../scripts/release-readiness-check.sh",
		"../scripts/reliability-gate.sh",
		"../scripts/perf-gate.sh",
		"../scripts/generate-sbom.sh",
		"../scripts/dependency-scan.sh",
		"../scripts/runtime-smoke.sh",
		"../scripts/verify-update-signature.sh",
		"../scripts/knit-codex-cli-adapter.sh",
		"../scripts/knit-claude-cli-adapter.sh",
		"../scripts/knit-opencode-cli-adapter.sh",
		"../scripts/knit-faster-whisper-stt.py",
	}
	for _, path := range files {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected script %s: %v", path, err)
		}
		if info.Mode().Perm()&0o100 == 0 {
			t.Fatalf("expected script %s to be executable", path)
		}
	}
}

func TestConfigExamplesExist(t *testing.T) {
	files := []string{
		"../.env.example",
		"../knit.toml.example",
	}
	for _, path := range files {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected config example %s: %v", path, err)
		}
	}

	raw, err := os.ReadFile("../.env.example")
	if err != nil {
		t.Fatalf("read .env.example: %v", err)
	}
	text := string(raw)
	required := []string{
		"# CODEX_MODEL=gpt-5-codex",
		"# KNIT_CLAUDE_API_MODEL=claude-3-7-sonnet-latest",
		"# OPENAI_STT_MODEL=gpt-4o-mini-transcribe",
		"# KNIT_TRANSCRIPTION_MODE=faster_whisper   # allowed: faster_whisper, local, lmstudio, remote",
		"# KNIT_CODEX_SANDBOX=workspace-write      # allowed: read-only, workspace-write, danger-full-access",
	}
	for _, fragment := range required {
		if !strings.Contains(text, fragment) {
			t.Fatalf("expected .env.example to include %q", fragment)
		}
	}
}

func TestGitHubActionsWorkflowReplacesGitLabCI(t *testing.T) {
	if _, err := os.Stat("../.github/workflows/ci.yml"); err != nil {
		t.Fatalf("expected GitHub Actions workflow: %v", err)
	}
	if _, err := os.Stat("../.github/workflows/release.yml"); err != nil {
		t.Fatalf("expected release workflow: %v", err)
	}
	if _, err := os.Stat("../LICENSE"); err != nil {
		t.Fatalf("expected repository license file: %v", err)
	}
	if _, err := os.Stat("../.gitlab-ci.yml"); !os.IsNotExist(err) {
		t.Fatalf("expected .gitlab-ci.yml to be removed, got err=%v", err)
	}
	raw, err := os.ReadFile("../.github/workflows/ci.yml")
	if err != nil {
		t.Fatalf("read GitHub Actions workflow: %v", err)
	}
	workflow := string(raw)
	required := []string{
		"name: CI",
		"branches:",
		"unit-tests:",
		"workflow-reliability-gate:",
		"performance-gate:",
		"build-matrix:",
		"package-release:",
		"sbom:",
		"vulnerability-scan:",
		"release-signature-verify:",
		"actions/setup-node@",
		"actions/setup-python@",
		"python3 -m pip install --upgrade build",
		"name: build-dist",
		"path: dist-ci/",
		"version=\"0.0.0-ci.${GITHUB_RUN_NUMBER}.${GITHUB_RUN_ATTEMPT}\"",
		"version=\"${GITHUB_REF_NAME#v}\"",
	}
	for _, fragment := range required {
		if !strings.Contains(workflow, fragment) {
			t.Fatalf("expected workflow fragment %q", fragment)
		}
	}
	disallowed := []string{
		"if: ${{ secrets.RELEASE_SIGNING_PRIVATE_KEY != '' }}",
		"if: ${{ secrets.RELEASE_SIGNING_PUBLIC_KEY != '' }}",
	}
	for _, fragment := range disallowed {
		if strings.Contains(workflow, fragment) {
			t.Fatalf("did not expect workflow to reference secrets directly in if expressions: %q", fragment)
		}
	}
	disallowedWorkflowFragments := []string{
		"publish-npm-main:",
		"create-github-release-main:",
		"npm publish --access public --tag main --provenance",
		"version=\"0.0.0-main.${GITHUB_RUN_NUMBER}.${GITHUB_RUN_ATTEMPT}\"",
	}
	for _, fragment := range disallowedWorkflowFragments {
		if strings.Contains(workflow, fragment) {
			t.Fatalf("did not expect CI workflow fragment %q", fragment)
		}
	}
	requiredEnvFragments := []string{
		"RELEASE_SIGNING_PRIVATE_KEY_SECRET: ${{ secrets.RELEASE_SIGNING_PRIVATE_KEY }}",
		"RELEASE_SIGNING_PUBLIC_KEY_SECRET: ${{ secrets.RELEASE_SIGNING_PUBLIC_KEY }}",
		"if: ${{ env.RELEASE_SIGNING_PRIVATE_KEY_SECRET != '' }}",
		"if: ${{ env.RELEASE_SIGNING_PUBLIC_KEY_SECRET != '' }}",
	}
	for _, fragment := range requiredEnvFragments {
		if !strings.Contains(workflow, fragment) {
			t.Fatalf("expected workflow fragment %q", fragment)
		}
	}

	releaseRaw, err := os.ReadFile("../.github/workflows/release.yml")
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}
	releaseWorkflow := string(releaseRaw)
	releaseRequired := []string{
		"name: Release",
		"workflow_dispatch:",
		"version:",
		"Release version must be a stable semantic version like 1.2.3.",
		"Releases must be cut from the main branch.",
		"Tag $tag already exists.",
		"dist-ci/bin missing before packaging; rebuilding release binaries",
		"npm publish --access public --tag latest --provenance dist-ci/packages/chadsly-knit-${VERSION}.tgz",
		"PYPI_API_TOKEN_SECRET: ${{ secrets.PYPI_API_TOKEN }}",
		"PYPI_API_TOKEN secret is required for stable PyPI releases.",
		"actions/setup-python@v5",
		"python3 -m pip install --upgrade build twine",
		"python3 -m twine upload \\",
		"dist-ci/packages/chadsly_knit-${VERSION}.tar.gz",
		"dist-ci/packages/chadsly_knit-${VERSION}-py3-none-any.whl",
		"softprops/action-gh-release@v2",
		"dist-ci/packages/*.tar.gz",
		"dist-ci/packages/*.zip",
		"dist-ci/packages/*.tgz",
		"dist-ci/packages/*.whl",
		"dist-ci/packages/installers/*.install.sh",
		"dist-ci/packages/installers/*.install.command",
		"dist-ci/packages/installers/*.install.ps1",
		"dist-ci/packages/checksums.txt",
		"dist-ci/packages/checksums.sig",
		"git tag -a \"$TAG\" -m \"Release $VERSION\"",
		"git push origin \"$TAG\"",
		"CI_COMMIT_TAG: ${{ steps.release.outputs.tag }}",
	}
	for _, fragment := range releaseRequired {
		if !strings.Contains(releaseWorkflow, fragment) {
			t.Fatalf("expected release workflow fragment %q", fragment)
		}
	}
	releaseDisallowed := []string{
		"dist-ci/packages/**",
		"packages/npm/knit-daemon",
		"packages/python/knit",
	}
	for _, fragment := range releaseDisallowed {
		if strings.Contains(releaseWorkflow, fragment) {
			t.Fatalf("did not expect release workflow fragment %q", fragment)
		}
	}
}

func TestCodexCLIAdapterUsesTopLevelApprovalFlagBeforeExec(t *testing.T) {
	raw, err := os.ReadFile("../scripts/knit-codex-cli-adapter.sh")
	if err != nil {
		t.Fatalf("read codex adapter script: %v", err)
	}
	script := string(raw)
	cmdIndex := strings.Index(script, `cmd=(codex)`)
	approvalIndex := strings.Index(script, `cmd+=(-a "$approval_policy")`)
	execIndex := strings.Index(script, `cmd+=(exec -C "$work_dir")`)
	if cmdIndex < 0 || approvalIndex < 0 || execIndex < 0 {
		t.Fatalf("expected codex adapter script to build top-level codex exec command with approval flag")
	}
	if !(cmdIndex < approvalIndex && approvalIndex < execIndex) {
		t.Fatalf("expected approval flag to be added before exec subcommand; got cmd=%d approval=%d exec=%d", cmdIndex, approvalIndex, execIndex)
	}
	if !strings.Contains(script, `instruction = str(payload.get("instruction_text") or "").strip()`) {
		t.Fatalf("expected codex adapter script to render shared instruction_text from the Knit CLI payload")
	}
	if strings.Contains(script, "Implement the requested software changes in the current repository.") {
		t.Fatalf("did not expect codex adapter script to hardcode an implementation-only prompt anymore")
	}
}

func TestBundledCLIAdapterScriptsUseSharedInstructionText(t *testing.T) {
	files := []string{
		"../scripts/knit-codex-cli-adapter.sh",
		"../scripts/knit-claude-cli-adapter.sh",
		"../scripts/knit-opencode-cli-adapter.sh",
	}
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read bundled cli adapter script %s: %v", path, err)
		}
		script := string(raw)
		if !strings.Contains(script, `instruction = str(payload.get("instruction_text") or "").strip()`) {
			t.Fatalf("expected %s to read instruction_text from the Knit CLI payload", path)
		}
		if !strings.Contains(script, "Do not ignore the selected delivery intent.") {
			t.Fatalf("expected %s to preserve the selected delivery intent in the rendered prompt", path)
		}
	}
}

func TestCoreRuntimePackagesAreGoPackages(t *testing.T) {
	packages := []string{
		"./cmd/daemon",
		"./cmd/ui",
		"./internal/server",
		"./internal/session",
		"./internal/agents",
		"./internal/security",
		"./internal/platform",
	}
	args := append([]string{"list", "-f", "{{.ImportPath}}:{{.Name}}"}, packages...)
	cmd := exec.Command("go", args...)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go list core packages failed: %v\n%s", err, string(out))
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != len(packages) {
		t.Fatalf("expected %d package rows, got %d: %q", len(packages), len(lines), string(out))
	}
	for _, line := range lines {
		if !strings.Contains(line, ":") {
			t.Fatalf("unexpected go list line %q", line)
		}
		parts := strings.SplitN(line, ":", 2)
		if parts[1] == "" {
			t.Fatalf("expected package name in %q", line)
		}
	}
}

func TestChromiumExtensionAssetsExist(t *testing.T) {
	files := []string{
		"../extension/chromium/manifest.json",
		"../extension/chromium/background.js",
		"../extension/chromium/icons/icon16.png",
		"../extension/chromium/icons/icon32.png",
		"../extension/chromium/icons/icon48.png",
		"../extension/chromium/icons/icon128.png",
		"../extension/chromium/popup.html",
		"../extension/chromium/popup.js",
		"../extension/chromium/recorder.html",
		"../extension/chromium/recorder.js",
	}
	for _, path := range files {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected extension asset %s: %v", path, err)
		}
	}
	raw, err := os.ReadFile("../extension/chromium/manifest.json")
	if err != nil {
		t.Fatalf("read extension manifest: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode extension manifest: %v", err)
	}
	if payload["manifest_version"] != float64(3) {
		t.Fatalf("expected mv3 manifest, got %#v", payload["manifest_version"])
	}
	icons, _ := payload["icons"].(map[string]any)
	if icons["128"] != "icons/icon128.png" {
		t.Fatalf("expected manifest 128 icon declaration, got %#v", icons["128"])
	}
	action, _ := payload["action"].(map[string]any)
	actionIcons, _ := action["default_icon"].(map[string]any)
	if actionIcons["32"] != "icons/icon32.png" {
		t.Fatalf("expected action icon declaration, got %#v", actionIcons["32"])
	}
}

func TestChromeWebStorePackagingAssetsExist(t *testing.T) {
	files := []string{
		"../docs/CHROME_WEB_STORE_LISTING.md",
		"../docs/PRIVACY_POLICY.md",
		"../docs/index.html",
		"../docs/assets/chrome-store/store-screenshot-1280x800.svg",
		"../docs/assets/chrome-store/store-screenshot-1280x800.png",
		"../docs/assets/chrome-store/promo-tile-440x280.svg",
		"../docs/assets/chrome-store/promo-tile-440x280.png",
		"../docs/assets/chrome-store/promo-marquee-1400x560.svg",
		"../docs/assets/chrome-store/promo-marquee-1400x560.png",
		"../docs/privacy-policy/index.html",
	}
	for _, path := range files {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected store asset %s: %v", path, err)
		}
		if info.Size() == 0 {
			t.Fatalf("expected store asset %s to be non-empty", path)
		}
	}
}

func TestChromiumExtensionAvoidsInlineEventHandlers(t *testing.T) {
	files := []string{
		"../extension/chromium/popup.html",
		"../extension/chromium/popup.js",
		"../extension/chromium/recorder.html",
		"../extension/chromium/recorder.js",
	}
	disallowed := []string{
		"onclick=",
		"onchange=",
		"oninput=",
		"onsubmit=",
		"onkeydown=",
		"javascript:",
	}
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read extension asset %s: %v", path, err)
		}
		content := strings.ToLower(string(raw))
		for _, fragment := range disallowed {
			if strings.Contains(content, fragment) {
				t.Fatalf("expected %s to avoid inline event handler fragment %q", path, fragment)
			}
		}
	}
}

func TestChromiumExtensionPopupIsCompactAndAccessible(t *testing.T) {
	raw, err := os.ReadFile("../extension/chromium/popup.html")
	if err != nil {
		t.Fatalf("read popup html: %v", err)
	}
	html := string(raw)
	if !strings.Contains(html, `min-width: 320px;`) {
		t.Fatalf("expected popup min width to be reduced for compact layout")
	}
	if strings.Contains(html, `id="composerCard"`) || strings.Contains(html, `id="textNoteWrap"`) {
		t.Fatalf("expected popup composer controls to move into the side panel")
	}
	if !strings.Contains(html, `id="openComposerBtn" class="primary"`) {
		t.Fatalf("expected popup launcher button for the side panel composer")
	}
	if !strings.Contains(html, `Capture, preview, and submit from the extension side panel after pairing.`) {
		t.Fatalf("expected popup copy to direct users into the side panel")
	}
	if !strings.Contains(html, `id="submitNotice" class="notice hidden"`) {
		t.Fatalf("expected submit notice region in popup")
	}
	if !strings.Contains(html, `id="unlinkBtn" class="danger"`) {
		t.Fatalf("expected popup to retain unpair action")
	}
}

func TestChromiumExtensionPopupJSIncludesCaptureFlows(t *testing.T) {
	raw, err := os.ReadFile("../extension/chromium/popup.js")
	if err != nil {
		t.Fatalf("read popup js: %v", err)
	}
	script := string(raw)
	required := []string{
		`consumeSubmitNotice`,
		`openComposerBtnEl`,
		`async function openComposerPanel()`,
		`type: "knit:bind-side-panel"`,
		`windowId: tab.windowId`,
		`chrome.sidePanel.open({ tabId: tab.id })`,
		`/api/extension/pair/complete`,
		`Extension paired. Opening the browser composer in the side panel.`,
		`type: "knit:consume-submit-notice"`,
		`The browser composer opened in the extension side panel for this tab.`,
	}
	for _, fragment := range required {
		if !strings.Contains(script, fragment) {
			t.Fatalf("expected popup script fragment %s", fragment)
		}
	}
}

func TestChromiumExtensionSidePanelHostsComposer(t *testing.T) {
	raw, err := os.ReadFile("../extension/chromium/recorder.html")
	if err != nil {
		t.Fatalf("read recorder html: %v", err)
	}
	html := string(raw)
	required := []string{
		`Browser Composer`,
		`html[data-theme="dark"] {`,
		`.theme-toggle {`,
		`.header-title-row {`,
		`.header-brand {`,
		`.header-mark {`,
		`src="icons/icon32.png" alt="Knit logo"`,
		`.header-status-row {`,
		`id="themeToggleBtn" class="theme-toggle"`,
		`title="Switch to dark theme"`,
		`id="sessionState" class="session-pill idle"`,
		`.tooltip-wrap {`,
		`.activity-state {`,
		`id="toggleTextBtn" class="icon-button"`,
		`id="snapshotBtn" class="icon-button"`,
		`id="audioBtn" class="icon-button"`,
		`id="videoBtn" class="icon-button"`,
		`id="previewBtn" class="icon-button"`,
		`id="submitBtn" class="icon-button"`,
		`id="stopBtn" class="icon-button"`,
		`id="textNoteWrap" class="hidden"`,
		`id="captureState" class="capture-state hidden"`,
		`id="activityState" class="preview-loading activity-state hidden"`,
		`id="preview" class="preview"`,
		`Preview updates here automatically as you capture notes, snapshots, audio, and video.`,
		`id="previewVideo" class="hidden"`,
		`data-tooltip="Type note. Show or hide the typed note field."`,
		`data-tooltip="Preview queued requests before submitting."`,
		`data-tooltip="Clear session."`,
		`.preview-note-header {`,
		`.preview-note-action {`,
		`.preview-media img {`,
		`.preview-media video {`,
		`.preview-spinner {`,
	}
	for _, fragment := range required {
		if !strings.Contains(html, fragment) {
			t.Fatalf("expected side panel html fragment %s", fragment)
		}
	}
	if strings.Contains(html, `id="startBtn" class="icon-button"`) {
		t.Fatalf("did not expect extension side panel to expose a start session button")
	}
}

func TestChromiumExtensionRecorderIncludesCaptureFlows(t *testing.T) {
	raw, err := os.ReadFile("../extension/chromium/recorder.js")
	if err != nil {
		t.Fatalf("read recorder js: %v", err)
	}
	script := string(raw)
	required := []string{
		`const PENDING_SNAPSHOT_KEY = "pendingSnapshotState";`,
		`const THEME_STORAGE_KEY = "sidePanelTheme";`,
		`const themeToggleBtnEl = document.getElementById("themeToggleBtn");`,
		`function normalizeTheme(theme)`,
		`function applyTheme(theme)`,
		`document.documentElement.setAttribute("data-theme", currentTheme);`,
		`storageGet([THEME_STORAGE_KEY])`,
		`storageSet({ [THEME_STORAGE_KEY]: nextTheme });`,
		`themeToggleBtnEl?.addEventListener("click", () => toggleTheme().catch((err) => setStatus(err.message, true)));`,
		`function setButtonTooltip(btn, tooltip)`,
		`btn.closest(".tooltip-wrap")`,
		`submitTypedNote`,
		`submitSnapshotNote`,
		`captureVisibleTabBlob`,
		`captureCurrentTabStream`,
		`chrome.tabs.captureVisibleTab`,
		`chrome.tabCapture.getMediaStreamId({ targetTabId: tab.id })`,
		`navigator.mediaDevices.getUserMedia`,
		`chromeMediaSource: "tab"`,
		`chromeMediaSourceId: id`,
		`const PREVIEW_EMPTY_HTML =`,
		`const PREVIEW_SUBMITTED_HTML =`,
		`let requestActivityMessage = "";`,
		`const PREVIEW_LOADING_HTML =`,
		`function renderActivityState()`,
		`captureStateEl.textContent = "";`,
		`captureStateEl.classList.add("hidden");`,
		`async function refreshPreviewAuto(options = {})`,
		`async function deletePreviewNote(eventID)`,
		`/api/session/feedback/delete`,
		`data-preview-action="delete"`,
		`previewEl.addEventListener("click", (event) => {`,
		`closest("[data-preview-action='delete']")`,
		`title="Remove queued request"`,
		`ICONS.trash`,
		`screenshot_data_url`,
		`video_data_url`,
		`URL.createObjectURL`,
		`data-preview-video-loading`,
		`setPreviewLoading("Loading preview…")`,
		`hydratePreviewMediaLoadState`,
		`postJSON("/api/companion/pointer"`,
		`postForm("/api/session/feedback/note"`,
		`postForm("/api/session/feedback/clip"`,
		`approveSession(true, "preview")`,
		`approveSession(true, "submit")`,
		`setRequestActivity("Starting microphone…")`,
		`setRequestActivity("Starting tab recording…")`,
		`setRequestActivity("Submitting request…")`,
		`previewSubmittedMessage = "Request submitted.";`,
		`setStatus("Request submitted.");`,
		`type: "knit:notify-submit"`,
		`Start a review session from the main UI first.`,
		`Snapshot queued. Type a note and press Cmd/Ctrl+Enter, or record audio or video next.`,
	}
	for _, fragment := range required {
		if !strings.Contains(script, fragment) {
			t.Fatalf("expected recorder script fragment %s", fragment)
		}
	}
	forbidden := []string{
		`async function startSession()`,
		`startBtnEl.addEventListener("click"`,
		`/api/session/start`,
	}
	for _, fragment := range forbidden {
		if strings.Contains(script, fragment) {
			t.Fatalf("did not expect recorder script fragment %s", fragment)
		}
	}
}

func TestChromiumExtensionManifestEnablesSidePanelAndTabCapture(t *testing.T) {
	raw, err := os.ReadFile("../extension/chromium/manifest.json")
	if err != nil {
		t.Fatalf("read extension manifest: %v", err)
	}
	var payload struct {
		Permissions []string       `json:"permissions"`
		SidePanel   map[string]any `json:"side_panel"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode extension manifest: %v", err)
	}
	if !strings.Contains(strings.Join(payload.Permissions, ","), "sidePanel") {
		t.Fatalf("expected sidePanel permission in manifest")
	}
	if !strings.Contains(strings.Join(payload.Permissions, ","), "tabCapture") {
		t.Fatalf("expected tabCapture permission in manifest")
	}
	if got := payload.SidePanel["default_path"]; got != "recorder.html" {
		t.Fatalf("expected side panel default path recorder.html, got %#v", got)
	}
}

func TestChromiumExtensionBackgroundTracksSubmitNotice(t *testing.T) {
	raw, err := os.ReadFile("../extension/chromium/background.js")
	if err != nil {
		t.Fatalf("read background js: %v", err)
	}
	script := string(raw)
	required := []string{
		`const SUBMIT_NOTICE_KEY = "lastSubmitNotice"`,
		`const BOUND_TAB_ID_KEY = "composerBoundTabId"`,
		`const BOUND_WINDOW_ID_KEY = "composerBoundWindowId"`,
		`async function applySidePanelBinding()`,
		`chrome.sidePanel.setOptions({ path: SIDE_PANEL_PATH, enabled: false })`,
		`chrome.sidePanel.setOptions({ tabId, path: SIDE_PANEL_PATH, enabled: true })`,
		`message.type === "knit:bind-side-panel"`,
		`message.type === "knit:clear-side-panel-binding"`,
		`chrome.tabs.onRemoved.addListener`,
		`message.type === "knit:notify-submit"`,
		`message.type === "knit:consume-submit-notice"`,
		`chrome.action.setBadgeText({ text: "1" })`,
		`chrome.storage.local.set({ [SUBMIT_NOTICE_KEY]: notice }`,
		`chrome.storage.local.remove([SUBMIT_NOTICE_KEY]`,
	}
	for _, fragment := range required {
		if !strings.Contains(script, fragment) {
			t.Fatalf("expected background script fragment %s", fragment)
		}
	}
}

func TestChromiumExtensionManifestDoesNotRequestMediaCapturePermissions(t *testing.T) {
	raw, err := os.ReadFile("../extension/chromium/manifest.json")
	if err != nil {
		t.Fatalf("read extension manifest: %v", err)
	}
	var payload struct {
		Permissions []string `json:"permissions"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode extension manifest: %v", err)
	}
	forbidden := []string{"audioCapture", "videoCapture", "desktopCapture"}
	for _, permission := range forbidden {
		for _, got := range payload.Permissions {
			if got == permission {
				t.Fatalf("did not expect extension manifest permission %s", permission)
			}
		}
	}
}
