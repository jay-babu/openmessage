//go:build !windows

package localapi

func isWindowsConnectionRefused(error) bool { return false }
