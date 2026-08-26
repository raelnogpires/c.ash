//go:build !windows

package storage

import "os"

func replaceAtomic(source, destination string) error {
	return os.Rename(source, destination)
}
