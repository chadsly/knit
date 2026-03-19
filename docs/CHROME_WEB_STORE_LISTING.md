# Chrome Web Store Listing

This file packages the current Chrome Web Store listing materials for the Knit extension.

## Upload Package

- Extension upload zip: `dist/knit-browser-composer-chrome-store.zip`
- 128x128 icon: `extension/chromium/icons/icon128.png`
- Store screenshot: `docs/assets/chrome-store/store-screenshot-1280x800.png`
- Small promo tile: `docs/assets/chrome-store/promo-tile-440x280.png`
- Marquee promo image: `docs/assets/chrome-store/promo-marquee-1400x560.png`

## Basic Info

### Name

Knit Browser Composer

### Short Description

Capture browser-grounded notes, snapshots, audio, and tab video for Knit without leaving the page.

### Detailed Description

Knit Browser Composer keeps browser review work attached to the page you are actually evaluating.

Use the extension side panel to:

- capture typed notes tied to the current page
- attach a snapshot of the active tab
- record a voice note
- record a short tab video with microphone audio
- preview queued requests before sending them to Knit
- submit browser-grounded feedback to your local Knit daemon without switching back to the main UI for every step

The extension is built for explicit, user-driven capture. It does not run background scraping or background recording. Pair it once with a local Knit daemon, open the side panel on the tab you want to review, and capture only what you choose to send.

### Category

Recommended: `Productivity`

Alternative if you want to position it more narrowly for engineering teams: `Developer Tools`

## Visuals

### Required

- 128x128 icon: `extension/chromium/icons/icon128.png`
- Screenshot: `docs/assets/chrome-store/store-screenshot-1280x800.png`

### Recommended

- Small promo tile: `docs/assets/chrome-store/promo-tile-440x280.png`
- Marquee promo image: `docs/assets/chrome-store/promo-marquee-1400x560.png`

## Privacy And Permissions

### Single Purpose Description

Use this for the Chrome Web Store "single purpose" field:

`Capture browser-grounded notes, snapshots, audio, and tab video from the current tab and send them to a local Knit daemon for review.`

### Privacy Practices Tab Copy

Use these justifications directly in the Chrome Web Store Privacy practices tab.

#### `activeTab` justification

`The extension uses activeTab only after the user opens the extension on the current page. This lets the extension bind the Browser Composer to that specific tab and collect page-specific context for the user's note, snapshot, audio, or tab-video request.`

#### Host permission justification

`The extension is limited to http://127.0.0.1/* and http://localhost/* so it can communicate with the user's local Knit daemon. It does not request arbitrary remote site access.`

#### Remote code use justification

`The extension does not load or execute remote code. All executable JavaScript, HTML, and CSS are packaged with the extension. Network access is used only to send user-requested data to the user's local Knit daemon over localhost.`

#### `scripting` justification

`The extension uses the scripting permission to read page context from the active tab after the user opens the Browser Composer. This context includes the page title, URL, selection, and focused-element details needed to create browser-grounded requests.`

#### `sidePanel` justification

`The extension uses the sidePanel permission to host the Browser Composer inside Chrome's side panel so the user can capture, preview, and submit notes without leaving the current page.`

#### `storage` justification

`The extension uses storage to keep the local daemon URL, the paired browser token, the tab binding for the side panel, and temporary queued snapshot state on the user's device.`

#### `tabCapture` justification

`The extension uses tabCapture only when the user explicitly starts a tab video recording. This records the current tab instead of the entire desktop.`

#### `tabs` justification

`The extension uses tabs to identify the active tab, bind the side panel to the correct page, and capture a visible-tab snapshot when the user explicitly requests one.`

### Permission Explanations

#### `activeTab`

Used to target the tab the user is actively reviewing after the user opens the extension. This is required to bind the side panel to the current page and to capture page-specific context only for that tab.

#### `sidePanel`

Used to host the full browser composer inside Chrome's side panel instead of forcing the user into a separate window.

#### `scripting`

Used to collect the current page title, URL, selection, and focused-element context after the user opens the composer. This powers browser-grounded note capture and preview metadata.

#### `storage`

Used to store the local daemon URL, the paired browser token, tab binding state, and short-lived queued snapshot state on the user's machine.

#### `tabCapture`

Used when the user explicitly chooses tab video capture so the extension can record the current tab, not the whole desktop.

#### `tabs`

Used to identify the active tab, bind the side panel to the correct tab, and capture the visible tab image when the user explicitly requests a snapshot.

#### `http://127.0.0.1/*` and `http://localhost/*`

Used so the extension can talk to the user's local Knit daemon. The extension does not require arbitrary remote host access.

### Privacy Policy

Because the extension handles user-authored notes, snapshots, and optional audio or video clips, publish a privacy policy alongside the Web Store listing.

Repository draft: `docs/PRIVACY_POLICY.md`

Before submitting to the store, host that policy at a stable public URL and place that URL in the Chrome Web Store listing.

## Privacy Practices Summary

Use the Chrome Web Store privacy section consistently with the actual extension behavior:

- Personally identifiable information:
  `Yes`
  User-authored notes, captured page content, snapshots, and tab video can include names, email addresses, account identifiers, or other identifiers depending on the page being reviewed.
- Health information:
  `No`
- Financial and payment information:
  `No`
- Authentication information:
  `Yes`
  The extension stores a local browser token used to authenticate to the local daemon.
- Personal communications:
  `Yes`
  User-authored typed notes and voice transcripts may contain message-like content entered by the user.
- Location:
  `No`
- Web history:
  `No`
  The extension does not collect browsing history in the background and does not maintain a visited-pages log.
- User activity:
  `No`
  The extension does not do click logging, scroll tracking, keystroke logging, or background monitoring. It only captures explicit user-requested note inputs and optional media capture.
- Website content:
  `Yes`
  The extension captures the current page URL, title, selection, focused-element context, and optional snapshots or tab video when the user explicitly triggers those actions.

For data usage declarations:

- Purpose: deliver the extension's user-facing note capture and submission workflow
- Sold to third parties: `No`
- Used for advertising: `No`
- Used for creditworthiness or lending decisions: `No`

### Dashboard Checklist

Recommended selections for the current extension behavior:

- Personally identifiable information: `Yes`
- Health information: `No`
- Financial and payment information: `No`
- Authentication information: `Yes`
- Personal communications: `Yes`
- Location: `No`
- Web history: `No`
- User activity: `No`
- Website content: `Yes`

Required certifications if the current implementation and deployment remain unchanged:

- `I do not sell or transfer user data to third parties, outside of the approved use cases`: check
- `I do not use or transfer user data for purposes that are unrelated to my item's single purpose`: check
- `I do not use or transfer user data to determine creditworthiness or for lending purposes`: check

### Privacy Policy URL

The Chrome Web Store requires a public URL here. A repository path is not enough.

Use a publicly reachable hosted page, for example:

- a GitHub Pages URL serving `docs/PRIVACY_POLICY.md`
- a docs site page under your product domain
- a support site page for the extension

Do not submit the item with a localhost URL, file path, or private document link.

### GitHub Pages Setup

This repository now includes a static Pages-ready privacy policy page at:

- `docs/privacy-policy/index.html`

The simplest setup is:

1. Push the repository to GitHub.
2. Open the repository on GitHub.
3. Go to `Settings` -> `Pages`.
4. Under `Build and deployment`, choose `Deploy from a branch`.
5. Select your default branch.
6. Select the `/docs` folder.
7. Save.

Once Pages finishes publishing, your URLs will typically be:

- Docs home:
  `https://YOUR-USER.github.io/YOUR-REPO/`
- Privacy policy:
  `https://YOUR-USER.github.io/YOUR-REPO/privacy-policy/`

Use the public `privacy-policy` URL in the Chrome Web Store listing.

### Compliance Items Outside The Repo

These are store account tasks, not code changes:

- Contact email:
  add a publisher contact email on the Chrome Web Store Account tab
- Contact email verification:
  complete the email verification flow on the Account tab before publishing
- Developer Program Policy certification:
  check the certification box on the Privacy practices tab before publishing

## Publish Notes

- Keep the listing text aligned with the actual permission use.
- Do not claim cloud sync or analytics unless you add them later.
- If the local daemon is configured to forward data to AI providers, disclose that in the product docs and ensure the hosted privacy policy matches the actual deployment configuration.

## Official References

- Chrome Web Store user data FAQ: https://developer.chrome.com/docs/webstore/program-policies/user-data-faq
- Chrome extension permission guidance: https://developer.chrome.com/docs/extensions/develop/concepts/declare-permissions
- Chrome Web Store branding page: https://developer.chrome.com/webstore/branding
