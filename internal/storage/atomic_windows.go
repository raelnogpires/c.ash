//go:build windows

package storage

import "golang.org/x/sys/windows"

func replaceAtomic(source, destination string) error {
	from, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(from, to, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}

// MoveFileEx with MOVEFILE_WRITE_THROUGH does not return until the replacement
// has been flushed. Opening the containing directory and calling File.Sync is
// not valid on Windows because os.Open creates a read-only directory handle.
func syncDirectory(string) error { return nil }
