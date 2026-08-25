//go:build !windows

package updater

import (
	"errors"
	"os"
	"syscall"
	"time"
)

func waitForProcess(pid int, timeout time.Duration) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		err = process.Signal(syscall.Signal(0))
		if errors.Is(err, os.ErrProcessDone) || errors.Is(err, syscall.ESRCH) {
			return nil
		}
		if err != nil && !errors.Is(err, syscall.EPERM) {
			return nil
		}
		time.Sleep(150 * time.Millisecond)
	}
	return errors.New("application did not close in time")
}
