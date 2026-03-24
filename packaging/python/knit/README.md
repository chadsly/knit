# `chadsly-knit`

`chadsly-knit` installs the host-specific Knit daemon from bundled release artifacts and exposes it through the `knit` command.

## Install

```bash
python3 -m pip install chadsly-knit
```

## Run

```bash
knit start
```

Install the browser extension after the daemon is running:

- Chrome Web Store: [Knit Browser Composer](https://chromewebstore.google.com/detail/knit-browser-composer/aepollbmimigbaapeelemgmdkfnhaclb?authuser=0&hl=en)
- Local unpacked install: open `chrome://extensions`, enable `Developer mode`, click `Load unpacked`, and select `extension/chromium` from the Knit repository

Pair it from the Knit UI with `Capture, review, and send -> Chrome Extension`.

## Commands

- `knit start`
  Ensures the packaged daemon is installed for the current host and starts it.
- `knit path`
  Prints the installed daemon binary path for the current host.
- `knit version`
  Prints the wrapper package version.

## How It Works

- The package bundles the released Knit archives for each supported target.
- On first use, the Python wrapper verifies the packaged archive checksum against the bundled release manifest.
- It extracts the host-specific daemon into the local user application-data directory and then runs it.

The daemon now prints the local UI URL, the Chrome Web Store link, the local unpacked extension path, and the pairing path in the Knit UI at startup. This package still only installs the daemon/runtime; the browser extension is installed separately.
