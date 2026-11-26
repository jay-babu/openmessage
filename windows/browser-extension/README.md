# Automatic Google login refresh on Windows

OpenMessage can use Microsoft Edge to refresh the login for an existing Google Messages pairing. When Google rejects saved credentials, OpenMessage reads a recent login from the local browser extension and reconnects using the existing phone pairing.

## Set up Edge once

1. Extract the Windows release archive to a folder you plan to keep. If building from source, run `python windows/build.py --output dist` with Go 1.26+ and Python 3 installed.
2. In PowerShell, run `.\register-browser-refresh.ps1` from the extracted or built folder. This registers the included native messaging host for your Windows user.
3. In the Edge profile signed in to the same Google account as your phone, open `edge://extensions`. Enable **Developer mode** if needed, choose **Load unpacked**, and select the included `browser-extension` folder. The badge shows **ON** when Edge has supplied a login. A **!** badge has a tooltip with the reason.
4. If OpenMessage already has a pairing, start it with `.\openmessage.exe serve`. For a new pairing, run `.\openmessage.exe pair --google-browser`, tap the matching emoji on your phone, then start the server.

Use OpenMessage's default Windows data directory and keep Edge running and signed in so it can supply fresh credentials. The extension reads the Google account cookies OpenMessage requires and the optional authentication cookies used by Google Messages. It sends them to the registered host on this PC. The host restricts the cached login to the first browser profile it receives and protects the cache with Windows Data Protection for the current user.

The local `/api/status` endpoint reports the Google connection and its latest credential-repair result.
