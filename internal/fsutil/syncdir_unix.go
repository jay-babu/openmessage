//go:build !windows

package fsutil

import (
	"errors"
	"os"
)

func SyncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}
