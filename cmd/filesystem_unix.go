//go:build !windows

package cmd

import (
	"errors"
	"fmt"
	"math"
	"os"
	"syscall"
)

func restrictFileCreation() func() {
	previous := syscall.Umask(0o077)
	return func() { syscall.Umask(previous) }
}

func lockInstanceFile(file *os.File) error {
	err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
		return fmt.Errorf("%w: %s", errInstanceLockHeld, file.Name())
	}
	return err
}

func filesystemAvailableBytes(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	blockSize := uint64(stat.Bsize)
	availableBlocks := uint64(stat.Bavail)
	if blockSize == 0 || availableBlocks > math.MaxUint64/blockSize {
		return 0, fmt.Errorf("filesystem free-space value overflows uint64")
	}
	return availableBlocks * blockSize, nil
}
