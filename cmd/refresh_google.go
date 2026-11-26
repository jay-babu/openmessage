package cmd

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/maxghenis/openmessage/internal/app"
	"github.com/maxghenis/openmessage/internal/googlecookies"
)

func RunRefreshGoogle(input io.Reader, args ...string) error {
	dataDir := app.DefaultDataDir()
	var cookies map[string]string
	var err error
	switch {
	case len(args) == 0:
		cookies, err = readRefreshCookies(input)
	case len(args) == 1 && args[0] == "--browser":
		cookies, err = googlecookies.LoadBrowser(dataDir)
	default:
		return fmt.Errorf("refresh-google accepts a Cookie header on stdin or --browser")
	}
	if err != nil {
		return err
	}

	deps := defaultPairDeps()
	if err := waitForBackendToReleaseSession(context.Background(), dataDir, deps.probe, deps.exitWait, deps.now, deps.sleep); err != nil {
		return err
	}
	sessionPath := filepath.Join(dataDir, "session.json")
	if err := googlecookies.UpdateSessionCookies(sessionPath, cookies); err != nil {
		return err
	}
	fmt.Println("Google login updated. Existing phone pairing kept.")
	return nil
}

func readRefreshCookies(input io.Reader) (map[string]string, error) {
	const maxInput = 64 * 1024
	raw, err := io.ReadAll(io.LimitReader(input, maxInput+1))
	if err != nil {
		return nil, fmt.Errorf("read Google Cookie header: %w", err)
	}
	if len(raw) > maxInput {
		return nil, fmt.Errorf("Google Cookie header is too large")
	}
	cookies, err := parseGoogleCookiesInput(string(raw))
	if err != nil {
		return nil, fmt.Errorf("paste a Cookie request header from a signed-in messages.google.com tab")
	}
	for _, name := range []string{"SID", "HSID", "SSID", "APISID", "SAPISID"} {
		if cookies[name] == "" {
			return nil, fmt.Errorf("Google Cookie header is missing %s. Copy the full header from a signed-in tab", name)
		}
	}

	return cookies, nil
}
