# Privacy Policy For Knit Browser Composer

Last updated: 2026-03-18

This privacy policy describes how the Knit Browser Composer Chrome extension handles data.

## Summary

The extension is designed for explicit, user-driven capture. It only collects page data, notes, snapshots, audio, or tab video when the user actively opens the extension and chooses a capture action.

The extension connects only to a user-configured local Knit daemon on `http://127.0.0.1` or `http://localhost`.

The extension does not include third-party advertising, analytics, or background browsing-history collection.

## What The Extension Collects

Depending on the action the user chooses, the extension may collect:

- the configured local daemon URL
- a browser token issued by the local daemon after pairing
- the current tab URL and page title
- the current text selection on the page
- limited focused-element context from the active page, such as element label, role, and a short text preview
- typed notes entered into the side panel
- page snapshots captured when the user clicks the snapshot button
- microphone audio captured when the user clicks the microphone button
- current-tab video and microphone audio captured when the user clicks the tab video button

The extension may temporarily store short-lived queued snapshot state in browser local storage so the user can attach that snapshot to the next typed, audio, or video note.

## How The Extension Uses Data

The extension uses collected data only to provide the user-facing capture workflow:

- pair the browser with the local daemon
- show the browser composer in the Chrome side panel
- prepare browser-grounded request previews
- send notes, snapshots, audio, and tab video to the local Knit daemon at the user's request

The extension does not use collected data for advertising, profiling, creditworthiness decisions, or sale to third parties.

## Where Data Goes

The extension sends captured data to the local Knit daemon configured by the user. The extension itself is limited to `http://127.0.0.1/*` and `http://localhost/*`.

The extension does not directly transmit data to third-party analytics or advertising services.

If the local Knit daemon is configured by the user or operator to send submitted material to other systems or AI providers, that downstream processing is controlled by the daemon environment, not by the extension alone.

## Data Retention

The extension stores a small amount of local state in Chrome local storage, including:

- daemon URL
- paired browser token
- tab-binding state
- temporary queued snapshot state

Captured notes and media are sent to the local daemon. Retention of submitted requests is controlled by the Knit daemon and its operator configuration.

## User Control

Users control when data is collected:

- no snapshot is captured until the user clicks the snapshot button
- no microphone audio is captured until the user clicks the microphone button
- no tab video is captured until the user clicks the tab video button
- users can clear a session from the side panel
- users can remove queued preview items before submitting them
- users can unpair the browser extension from the local daemon

## Data Sharing

The extension does not sell user data.

The extension does not share user data with advertisers.

The extension shares captured data only with the user-configured local daemon needed to deliver the extension's core functionality.

## Security

The extension stores its local settings using Chrome extension storage and authenticates to the local daemon with the paired browser token issued by that daemon.

Because the extension can capture page context and optional media, users should pair it only with a daemon they trust and should review the page they are capturing before submitting.

## Contact

For operator or publisher questions, provide the support or contact URL used for the Knit deployment.
