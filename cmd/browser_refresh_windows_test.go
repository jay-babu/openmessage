package cmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/maxghenis/openmessage/internal/googlecookies"
)

func TestWindowsBrowserCredentialRecovery(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("OPENMESSAGES_DATA_DIR", dataDir)
	t.Setenv("OPENMESSAGE_COOKIE_REFRESH_SCRIPT", "")
	sessionPath := filepath.Join(dataDir, "session.json")
	const original = `{"auth_data":{"device_number":9007199254740999,"pairing_id":"existing","cookies":{}}}`
	if err := os.WriteFile(sessionPath, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	if canRefreshGoogleCookies() {
		t.Fatal("recovery is available before a browser login has been registered")
	}
	for _, token := range []string{"test-login-1", "test-login-2"} {
		cookies := map[string]string{"SID": token, "HSID": "test-hsid", "SSID": "test-ssid", "APISID": "test-apisid", "SAPISID": "test-sapisid", "__Secure-1PSIDTS": "test-session-timestamp", "__Secure-OSID": "test-secure-service"}
		if err := googlecookies.SaveBrowserLogin(dataDir, "test-profile", cookies); err != nil {
			t.Fatal(err)
		}
		if !canRefreshGoogleCookies() {
			t.Fatal("recovery is unavailable after registering the browser login")
		}
		if err := refreshGoogleSessionCookies(context.Background(), sessionPath); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(sessionPath)
		if err != nil {
			t.Fatal(err)
		}
		var session struct {
			Auth struct {
				DeviceNumber int64             `json:"device_number"`
				PairingID    string            `json:"pairing_id"`
				Cookies      map[string]string `json:"cookies"`
			} `json:"auth_data"`
		}
		if err := json.Unmarshal(data, &session); err != nil {
			t.Fatal(err)
		}
		if session.Auth.PairingID != "existing" || session.Auth.DeviceNumber != 9007199254740999 || session.Auth.Cookies["SID"] != token ||
			session.Auth.Cookies["__Secure-1PSIDTS"] != "test-session-timestamp" ||
			session.Auth.Cookies["__Secure-OSID"] != "test-secure-service" {
			t.Fatal("credential recovery did not replace only the cookies")
		}
	}
}
