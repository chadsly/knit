# Knit Daemon npm Package

This package installs the host-specific Knit daemon and UI binaries from the packaged release artifacts bundled inside the package.

Commands:

- `npx knit-daemon start`
- `npx knit-daemon path`
- `npx knit-daemon version`

The install step verifies the selected archive against the bundled release manifest before extraction.
