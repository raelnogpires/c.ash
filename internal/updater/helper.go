package updater

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

// RunHelper applies a previously verified package after the desktop app exits.
// It is invoked only by the installed application, never by the frontend.
func RunHelper(args []string) error {
	fs := flag.NewFlagSet("cash-updater", flag.ContinueOnError)
	parentPID := fs.Int("parent-pid", 0, "parent process ID")
	packagePath := fs.String("package", "", "verified package")
	appPath := fs.String("app", "", "application executable")
	resultPath := fs.String("result", "", "result marker")
	platform := fs.String("platform", "", "target platform")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *parentPID <= 0 || *packagePath == "" || *appPath == "" || *resultPath == "" || (*platform != "windows" && *platform != "linux") {
		return errors.New("invalid updater arguments")
	}
	if err := waitForProcess(*parentPID, 45*time.Second); err != nil {
		return err
	}
	err := applyPackage(*platform, *packagePath)
	result := struct {
		Success bool   `json:"success"`
		Message string `json:"message,omitempty"`
	}{Success: err == nil}
	if err != nil {
		result.Message = "A atualização não foi instalada. A versão anterior foi mantida."
	}
	_ = writeJSON(*resultPath, result)
	if restartErr := exec.Command(*appPath).Start(); restartErr != nil && err == nil {
		return fmt.Errorf("restart application: %w", restartErr)
	}
	return err
}

func applyPackage(platform, packagePath string) error {
	if _, err := os.Stat(packagePath); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	cmd, err := installCommand(ctx, platform, packagePath)
	if err != nil {
		return err
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("apply update: %w (%s)", err, string(output))
	}
	return nil
}

func installCommand(ctx context.Context, platform, packagePath string) (*exec.Cmd, error) {
	switch platform {
	case "windows":
		return exec.CommandContext(ctx, packagePath, "/S"), nil
	case "linux":
		if filepath.Ext(packagePath) != ".deb" {
			return nil, errors.New("linux update is not a deb package")
		}
		return exec.CommandContext(ctx, "pkexec", "/usr/bin/apt-get", "install", "--yes", packagePath), nil
	default:
		return nil, errors.New("unsupported update platform")
	}
}

// ParseHelperArguments is intentionally tiny but useful to platform smoke tests.
func ParseHelperArguments(args []string) (int, error) {
	if len(args) < 2 {
		return 0, errors.New("missing process ID")
	}
	return strconv.Atoi(args[1])
}
