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

When you run `knit start`, the native daemon prints the local UI URL. Knit’s browser extension and main review UI still live outside this npm wrapper; this package is only the daemon/runtime install path.
