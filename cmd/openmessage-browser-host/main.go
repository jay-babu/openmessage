//go:build windows

package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/maxghenis/openmessage/internal/googlecookies"
)

// Set from the extension's public key when building the native host.
var allowedOrigin string

func main() {
	if allowedOrigin == "" || len(os.Args) < 2 || os.Args[1] != allowedOrigin {
		os.Exit(1)
	}
	err := receiveLogin(os.Stdin)
	result := struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}{OK: err == nil}
	if err != nil {
		result.Error = err.Error()
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		os.Exit(1)
	}
	if err := binary.Write(os.Stdout, binary.LittleEndian, uint32(len(encoded))); err != nil {
		os.Exit(1)
	}
	if _, err := os.Stdout.Write(encoded); err != nil {
		os.Exit(1)
	}
}

func receiveLogin(input io.Reader) error {
	var size uint32
	if err := binary.Read(input, binary.LittleEndian, &size); err != nil || size == 0 || size > 32*1024 {
		return errors.New("invalid native message size")
	}
	data := make([]byte, size)
	defer clear(data)
	if _, err := io.ReadFull(input, data); err != nil {
		return errors.New("could not read native message")
	}
	var request struct {
		Profile string            `json:"profile"`
		Cookies map[string]string `json:"cookies"`
	}
	if err := json.Unmarshal(data, &request); err != nil {
		return errors.New("invalid native login message")
	}
	root := os.Getenv("LOCALAPPDATA")
	if !filepath.IsAbs(root) {
		return errors.New("Windows local app data directory is unavailable")
	}
	return googlecookies.SaveBrowserLogin(filepath.Join(root, "OpenMessage"), request.Profile, request.Cookies)
}
