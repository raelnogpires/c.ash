package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"runtime"

	frontendassets "c.ash/frontend"
	"c.ash/internal/application"
	"c.ash/internal/desktop"
	"c.ash/internal/domain"
	"c.ash/internal/storage"
	"c.ash/internal/updater"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

// These values are set by the release workflow. Development builds remain
// deliberately unable to download or install updates.
var (
	version    = "dev"
	commit     = ""
	repository = ""
)

func main() {
	configureWebView()
	appContext := context.Background()
	path, err := storage.DefaultPath()
	if err != nil {
		log.Fatal(err)
	}
	store, err := storage.OpenWithVersion(appContext, path, version)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	theme := domain.ThemeLight
	if !store.SecurityStatus().Locked {
		profile, profileErr := store.Profile(appContext)
		if profileErr != nil {
			log.Printf("read saved theme for window icon: %v", profileErr)
		} else if profile != nil && domain.ValidateTheme(profile.Theme) == nil {
			theme = profile.Theme
		}
	}
	helperName := "cash-updater"
	if runtime.GOOS == "windows" {
		helperName += ".exe"
	}
	executablePath, executableErr := os.Executable()
	if executableErr != nil {
		executablePath = os.Args[0]
	}
	updaterManager, updateErr := updater.New(updater.BuildInfo{Version: version, Commit: commit, Repository: repository}, updater.Options{HelperPath: filepath.Join(filepath.Dir(executablePath), helperName), Executable: executablePath})
	if updateErr != nil {
		log.Printf("initialize updater: %v", updateErr)
	}
	app := desktop.New(application.New(store, nil), updaterManager, func(theme domain.Theme) {
		setPlatformIcon(nativeIcon(theme))
	})
	app.SetVersion(version)
	err = wails.Run(&options.App{
		Title: "[c]ash", Width: 1180, Height: 760, MinWidth: 960, MinHeight: 640,
		AssetServer:      &assetserver.Options{Assets: frontendassets.Assets},
		OnStartup:        app.Startup,
		Bind:             []interface{}{app},
		BackgroundColour: &options.RGBA{R: 247, G: 245, B: 239, A: 1},
		Linux:            platformOptions(nativeIcon(theme)),
	})
	if err != nil {
		log.Fatal(err)
	}
}
