# `@chadsly/knit`

`@chadsly/knit` installs the host-specific Knit daemon from bundled release artifacts and exposes it through the `knit` command.

Install:

```bash
npm install -g @chadsly/knit
```

Start Knit:

```bash
knit start
```

Install the browser extension after the daemon is running:

- Chrome Web Store: [Knit Browser Composer](https://chromewebstore.google.com/detail/knit-browser-composer/aepollbmimigbaapeelemgmdkfnhaclb?authuser=0&hl=en)
- Local unpacked install: open `chrome://extensions`, enable `Developer mode`, click `Load unpacked`, and select `extension/chromium` from the Knit repository

Pair it from the Knit UI with `Capture, review, and send -> Chrome Extension`.

Other commands:

- `knit path`
- `knit version`

What happens on install:

- `postinstall` selects the archive for the current OS and CPU
- verifies it against the bundled release manifest
- extracts the native daemon into the package runtime directory

Supported platforms:

- macOS: `x64`, `arm64`
- Linux: `x64`, `arm64`
- Windows: `x64`

When you run `knit start`, the native daemon prints the local UI URL, the Chrome Web Store link, the local unpacked extension path, and the pairing path in the Knit UI. This package still only installs the daemon/runtime; the browser extension is installed separately.
