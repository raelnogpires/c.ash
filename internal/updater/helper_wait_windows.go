//go:build windows

package updater

import (
	"errors"
	"time"

	"golang.org/x/sys/windows"
)

func waitForProcess(pid int, timeout time.Duration) error {
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return nil
	} // Parent already exited.
	defer windows.CloseHandle(handle)
	result, err := windows.WaitForSingleObject(handle, uint32(timeout.Milliseconds()))
	if err != nil {
		return err
	}
	if result != windows.WAIT_OBJECT_0 {
		return errors.New("application did not close in time")
	}
	return nil
}
