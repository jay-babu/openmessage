package googlecookies

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const browserCredentialFile = "google-browser-login.bin"
const browserCredentialMaxAge = 5 * time.Minute

type browserLogin struct {
	Profile string            `json:"profile"`
	Cookies map[string]string `json:"cookies"`
}

func NativeSupported() bool { return false }

func defaultChromeProfileDir(home string) string {
	return filepath.Join(home, "AppData", "Local", "Google", "Chrome", "User Data", "Default")
}

func Load(context.Context, string) (map[string]string, error) {
	return nil, errors.New("use the OpenMessage browser extension on Windows")
}

func loadForRefresh(_ context.Context, _ string, sessionPath string) (map[string]string, error) {
	return LoadBrowser(filepath.Dir(sessionPath))
}

func BrowserAvailable(dataDir string) bool {
	if _, err := os.Stat(filepath.Join(dataDir, browserCredentialFile)); err == nil {
		return true
	}
	// Keep retrying during first-time extension setup so recovery can
	// start as soon as the linked browser supplies the Google login.
	localAppData := os.Getenv("LOCALAPPDATA")
	if !filepath.IsAbs(localAppData) || !strings.EqualFold(filepath.Clean(dataDir), filepath.Join(localAppData, "OpenMessage")) {
		return false
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, "Software\\Microsoft\\Edge\\NativeMessagingHosts\\com.openmessage.google_refresh", registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	manifest, _, err := key.GetStringValue("")
	return err == nil && manifest != ""
}

func LoadBrowser(dataDir string) (map[string]string, error) {
	login, updatedAt, err := readBrowserLogin(filepath.Join(dataDir, browserCredentialFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("OpenMessage has not received a login from the browser extension")
	}
	if err != nil {
		return nil, err
	}
	if time.Since(updatedAt) > browserCredentialMaxAge {
		return nil, errors.New("browser login is stale; keep the signed-in browser running")
	}
	return requiredBrowserCookies(login.Cookies)
}

func SaveBrowserLogin(dataDir, profile string, cookies map[string]string) error {
	if profile == "" || len(profile) > 80 {
		return errors.New("invalid browser profile ID")
	}
	var err error
	if cookies != nil {
		cookies, err = requiredBrowserCookies(cookies)
		if err != nil {
			return err
		}
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(dataDir, browserCredentialFile)
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer lock.Close()
	var offset windows.Overlapped
	if err := windows.LockFileEx(windows.Handle(lock.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, &offset); err != nil {
		return err
	}
	if previous, _, err := readBrowserLogin(path); err == nil {
		if previous.Profile != profile {
			return errors.New("a different browser profile already supplies the OpenMessage login")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	plain, err := json.Marshal(browserLogin{Profile: profile, Cookies: cookies})
	if err != nil {
		return errors.New("could not encode browser login")
	}
	defer clear(plain)
	sealed, err := protectBrowserLogin(plain, true)
	if err != nil {
		return fmt.Errorf("protect browser login: %w", err)
	}
	return writeBrowserLogin(path, sealed)
}

func readBrowserLogin(path string) (browserLogin, time.Time, error) {
	file, err := os.Open(path)
	if err != nil {
		return browserLogin{}, time.Time{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return browserLogin{}, time.Time{}, err
	}
	if info.Size() == 0 || info.Size() > 64*1024 {
		return browserLogin{}, time.Time{}, errors.New("invalid protected browser login size")
	}
	sealed := make([]byte, info.Size())
	if _, err := io.ReadFull(file, sealed); err != nil {
		return browserLogin{}, time.Time{}, err
	}
	plain, err := protectBrowserLogin(sealed, false)
	if err != nil {
		return browserLogin{}, time.Time{}, fmt.Errorf("read protected browser login: %w", err)
	}
	defer clear(plain)
	var login browserLogin
	if err := json.Unmarshal(plain, &login); err != nil || login.Profile == "" {
		return browserLogin{}, time.Time{}, errors.New("invalid protected browser login")
	}
	return login, info.ModTime(), nil
}

func requiredBrowserCookies(cookies map[string]string) (map[string]string, error) {
	if cookies == nil {
		return nil, errors.New("sign in to Google Messages in the linked browser profile")
	}
	selected := make(map[string]string, len(requiredCookies)+3)
	for _, required := range requiredCookies {
		value := cookies[required.name]
		if value == "" || len(value) > 4096 {
			return nil, fmt.Errorf("browser login is missing a usable %s", required.name)
		}
		selected[required.name] = value
	}
	// Some Google sessions require the account timestamp or the secure
	// Messages service cookie in addition to the five account cookies.
	for _, name := range []string{"OSID", "__Secure-OSID", "__Secure-1PSIDTS"} {
		if value := cookies[name]; value != "" && len(value) <= 4096 {
			selected[name] = value
		}
	}
	return selected, nil
}

func writeBrowserLogin(path string, sealed []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".browser-login-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, writeErr := file.Write(sealed)
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return err
	}
	source, err := windows.UTF16PtrFromString(file.Name())
	if err != nil {
		return err
	}
	target, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(source, target, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}

func protectBrowserLogin(data []byte, encrypt bool) ([]byte, error) {
	in := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	var out windows.DataBlob
	var err error
	const noUI = windows.CRYPTPROTECT_UI_FORBIDDEN
	if encrypt {
		err = windows.CryptProtectData(&in, nil, nil, 0, nil, noUI, &out)
	} else {
		err = windows.CryptUnprotectData(&in, nil, nil, 0, nil, noUI, &out)
	}
	if err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(uintptr(unsafe.Pointer(out.Data))))
	result := unsafe.Slice(out.Data, int(out.Size))
	defer clear(result)
	return append([]byte(nil), result...), nil
}
