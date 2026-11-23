package cmd

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// Windows files inherit the access control list of their parent directory.
func restrictFileCreation() func() {
	return func() {}
}

func lockInstanceFile(file *os.File) error {
	// Lock a byte beyond EOF so another process can still read the diagnostic
	// record. Windows releases the byte-range lock when the handle closes.
	offset := windows.Overlapped{OffsetHigh: 1}
	flags := uint32(windows.LOCKFILE_EXCLUSIVE_LOCK | windows.LOCKFILE_FAIL_IMMEDIATELY)
	err := windows.LockFileEx(windows.Handle(file.Fd()), flags, 0, 1, 0, &offset)
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return fmt.Errorf("%w: %s", errInstanceLockHeld, file.Name())
	}
	return err
}

func filesystemAvailableBytes(path string) (uint64, error) {
	directory, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var available uint64
	err = windows.GetDiskFreeSpaceEx(directory, &available, nil, nil)
	return available, err
}
