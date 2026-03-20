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

The daemon still prints the local UI URL at startup. Knit's browser extension and main review UI remain separate from this PyPI wrapper; this package is only the daemon/runtime install path.
