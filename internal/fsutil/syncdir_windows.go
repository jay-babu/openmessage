package fsutil

// Windows cannot flush an ordinary directory handle. Callers still flush file
// contents before publishing the directory entry.
func SyncDir(string) error { return nil }
