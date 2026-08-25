package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestCheck_NewStableReleaseIsAvailable(t *testing.T) {
	payload := []byte("cash installer")
	digest := sha256.Sum256(payload)
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/cash/releases/latest" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		fmt.Fprintf(w, `{"tag_name":"v0.2.0","body":"Correções","published_at":"2026-08-21T12:00:00Z","assets":[{"name":"cash_0.2.0_linux_amd64.deb","size":%d,"digest":"sha256:%s","browser_download_url":"%s/download"}]}`, len(payload), hex.EncodeToString(digest[:]), server.URL)
	}))
	defer server.Close()
	manager := testManager(t, server.URL, "0.1.0")
	status, err := manager.Check(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if status.State != Available || status.AvailableVersion != "0.2.0" || status.ReleaseNotes != "Correções" {
		t.Fatalf("status=%+v", status)
	}
}

func TestCheck_DailyAutomaticCheckIsThrottled(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { requests.Add(1); w.WriteHeader(http.StatusNotFound) }))
	defer server.Close()
	manager := testManager(t, server.URL, "0.1.0")
	if _, err := manager.Check(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Check(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("requests=%d, want 1", got)
	}
}

func TestCheck_RejectsMissingDigest(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"tag_name":"v0.2.0","assets":[{"name":"cash_0.2.0_linux_amd64.deb","size":12,"browser_download_url":"%s/download"}]}`, server.URL)
	}))
	defer server.Close()
	manager := testManager(t, server.URL, "0.1.0")
	status, err := manager.Check(context.Background(), true)
	if err == nil {
		t.Fatal("expected missing digest error")
	}
	if status.State != Failed {
		t.Fatalf("status=%+v", status)
	}
}

func TestInstall_RejectsDigestMismatchBeforeStartingHelper(t *testing.T) {
	payload := []byte("untrusted package")
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/acme/cash/releases/latest":
			fmt.Fprintf(w, `{"tag_name":"v0.2.0","assets":[{"name":"cash_0.2.0_linux_amd64.deb","size":%d,"digest":"sha256:%064d","browser_download_url":"%s/download"}]}`, len(payload), 0, server.URL)
		case "/download":
			_, _ = w.Write(payload)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	manager := testManager(t, server.URL, "0.1.0")
	if _, err := manager.Check(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	status, err := manager.Install(context.Background())
	if err == nil {
		t.Fatal("expected digest mismatch")
	}
	if status.State != Failed {
		t.Fatalf("status=%+v", status)
	}
}

func TestInstallCommand_UsesPlatformSpecificArguments(t *testing.T) {
	linux, err := installCommand(context.Background(), "linux", "/tmp/cash.deb")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := linux.Args, []string{"pkexec", "/usr/bin/apt-get", "install", "--yes", "/tmp/cash.deb"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("linux args=%v", got)
	}
	windows, err := installCommand(context.Background(), "windows", `C:\\update.exe`)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := windows.Args, []string{`C:\\update.exe`, "/S"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("windows args=%v", got)
	}
}

func testManager(t *testing.T, endpoint, version string) *Manager {
	t.Helper()
	dir := t.TempDir()
	helper := filepath.Join(dir, "cash-updater")
	if err := os.WriteFile(helper, []byte("placeholder"), 0o700); err != nil {
		t.Fatal(err)
	}
	manager, err := New(BuildInfo{Version: version, Repository: "acme/cash"}, Options{ConfigDir: filepath.Join(dir, "config"), CacheDir: filepath.Join(dir, "cache"), HelperPath: helper, Executable: filepath.Join(dir, "cash"), APIBaseURL: endpoint, AllowHTTP: true, GOOS: "linux", GOARCH: "amd64"})
	if err != nil {
		t.Fatal(err)
	}
	return manager
}
