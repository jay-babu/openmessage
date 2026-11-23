package localapi

import (
	"errors"

	"golang.org/x/sys/windows"
)

func isWindowsConnectionRefused(err error) bool {
	return errors.Is(err, windows.WSAECONNREFUSED)
}
