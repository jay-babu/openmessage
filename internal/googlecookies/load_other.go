//go:build !windows

package googlecookies

import (
	"context"
	"fmt"
)

func loadForRefresh(ctx context.Context, profile, _ string) (map[string]string, error) {
	return Load(ctx, profile)
}

func LoadBrowser(string) (map[string]string, error) {
	return nil, fmt.Errorf("the OpenMessage browser extension is available on Windows")
}

// Load reads and decrypts the Google cookies of a Chrome profile with the
// Chrome Safe Storage secret from the login keychain. It is what Refresh
// writes into session.json and what `pair --google-chrome` pairs with.
// Callers must never log the returned values.
func Load(ctx context.Context, profile string) (map[string]string, error) {
	if profile == "" {
		return nil, fmt.Errorf("no Chrome profile directory")
	}
	secret, err := chromeSafeStorageSecret(ctx)
	if err != nil {
		return nil, fmt.Errorf("chrome safe storage secret: %w", err)
	}
	return LoadChromeCookies(profile, secret)
}

func BrowserAvailable(string) bool { return false }
