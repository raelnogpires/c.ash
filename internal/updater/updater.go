// Package updater checks, downloads and hands off signed-by-digest application
// releases. It deliberately has no dependency on the financial domain.
package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	checkInterval = 24 * time.Hour
	maxDownload   = int64(250 * 1024 * 1024)
)

var versionPattern = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// BuildInfo is filled by the release build through -ldflags.
type BuildInfo struct {
	Version    string
	Commit     string
	Repository string
}

// State is the user-visible updater state.
type State string

const (
	Disabled    State = "disabled"
	Idle        State = "idle"
	Checking    State = "checking"
	UpToDate    State = "upToDate"
	Available   State = "available"
	Downloading State = "downloading"
	Installing  State = "installing"
	Failed      State = "error"
)

// Status is safe to expose through the Wails bridge.
type Status struct {
	State            State  `json:"state"`
	CurrentVersion   string `json:"currentVersion"`
	AvailableVersion string `json:"availableVersion,omitempty"`
	ReleaseNotes     string `json:"releaseNotes,omitempty"`
	PublishedAt      string `json:"publishedAt,omitempty"`
	LastCheckedAt    string `json:"lastCheckedAt,omitempty"`
	DownloadedBytes  int64  `json:"downloadedBytes,omitempty"`
	TotalBytes       int64  `json:"totalBytes,omitempty"`
	Message          string `json:"message,omitempty"`
}

// Options makes paths and the GitHub endpoint injectable in tests. APIBaseURL
// must be HTTPS in production.
type Options struct {
	ConfigDir  string
	CacheDir   string
	HelperPath string
	Executable string
	APIBaseURL string
	HTTPClient *http.Client
	Now        func() time.Time
	GOOS       string
	GOARCH     string
	AllowHTTP  bool // tests only
}

type persistedState struct {
	LastCheckedAt string `json:"lastCheckedAt,omitempty"`
	ETag          string `json:"etag,omitempty"`
}

type release struct {
	TagName     string  `json:"tag_name"`
	Body        string  `json:"body"`
	PublishedAt string  `json:"published_at"`
	Draft       bool    `json:"draft"`
	Prerelease  bool    `json:"prerelease"`
	Assets      []asset `json:"assets"`
}

type asset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	Digest             string `json:"digest"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type releaseAsset struct {
	Version string
	Notes   string
	Date    string
	Asset   asset
}

// Manager owns update state for one process.
type Manager struct {
	checkMu     sync.Mutex
	mu          sync.RWMutex
	info        BuildInfo
	options     Options
	status      Status
	persisted   persistedState
	pending     *releaseAsset
	listeners   []func(Status)
	statePath   string
	resultPath  string
	downloadDir string
}

// New returns a disabled manager when the build is not a released supported
// binary. That makes local development and unsupported platforms safe by default.
func New(info BuildInfo, options Options) (*Manager, error) {
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.HTTPClient == nil {
		options.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	if options.GOOS == "" {
		options.GOOS = runtime.GOOS
	}
	if options.GOARCH == "" {
		options.GOARCH = runtime.GOARCH
	}
	if options.APIBaseURL == "" {
		options.APIBaseURL = "https://api.github.com"
	}
	if options.Executable == "" {
		var err error
		options.Executable, err = os.Executable()
		if err != nil {
			return nil, fmt.Errorf("find executable: %w", err)
		}
	}
	if options.ConfigDir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("find config directory: %w", err)
		}
		options.ConfigDir = filepath.Join(base, "c.ash")
	}
	if options.CacheDir == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			return nil, fmt.Errorf("find cache directory: %w", err)
		}
		options.CacheDir = filepath.Join(base, "c.ash", "updates")
	}
	m := &Manager{
		info: info, options: options,
		status:      Status{State: Disabled, CurrentVersion: info.Version},
		statePath:   filepath.Join(options.ConfigDir, "update-state.json"),
		resultPath:  filepath.Join(options.ConfigDir, "update-result.json"),
		downloadDir: options.CacheDir,
	}
	if m.configured() {
		m.status.State = Idle
	}
	m.loadPersisted()
	m.loadResult()
	return m, nil
}

func (m *Manager) configured() bool {
	if !validVersion(m.info.Version) || strings.TrimSpace(m.info.Repository) == "" {
		return false
	}
	if m.options.GOARCH != "amd64" || (m.options.GOOS != "windows" && m.options.GOOS != "linux") {
		return false
	}
	u, err := url.Parse(m.options.APIBaseURL)
	return err == nil && u.Host != "" && (m.options.AllowHTTP || u.Scheme == "https")
}

// Subscribe receives each state transition. The callback is invoked outside the
// manager lock and should return quickly.
func (m *Manager) Subscribe(listener func(Status)) {
	if listener == nil {
		return
	}
	m.mu.Lock()
	m.listeners = append(m.listeners, listener)
	status := m.status
	m.mu.Unlock()
	listener(status)
}

// Status returns a copy of the current state.
func (m *Manager) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

// Check checks the latest stable GitHub release. Manual checks bypass the daily
// throttle and surface connection errors; automatic checks remain quiet.
func (m *Manager) Check(ctx context.Context, manual bool) (Status, error) {
	m.checkMu.Lock()
	defer m.checkMu.Unlock()
	if !m.configured() {
		return m.Status(), nil
	}
	m.mu.RLock()
	last := parseTime(m.persisted.LastCheckedAt)
	m.mu.RUnlock()
	if !manual && !last.IsZero() && m.options.Now().Sub(last) < checkInterval {
		return m.Status(), nil
	}
	m.setStatus(Status{State: Checking, CurrentVersion: m.info.Version, LastCheckedAt: m.persisted.LastCheckedAt})

	rel, etag, notModified, err := m.latest(ctx)
	m.recordCheck(etag)
	if err != nil {
		if manual {
			m.setStatus(Status{State: Failed, CurrentVersion: m.info.Version, LastCheckedAt: m.persisted.LastCheckedAt, Message: "Não foi possível verificar atualizações. Confira sua conexão e tente novamente."})
			return m.Status(), err
		}
		m.setStatus(Status{State: Idle, CurrentVersion: m.info.Version, LastCheckedAt: m.persisted.LastCheckedAt})
		return m.Status(), nil
	}
	if notModified || rel == nil || rel.Draft || rel.Prerelease || !validVersion(rel.TagName) || compareVersions(rel.TagName, m.info.Version) <= 0 {
		m.mu.Lock()
		m.pending = nil
		m.mu.Unlock()
		m.setStatus(Status{State: UpToDate, CurrentVersion: m.info.Version, LastCheckedAt: m.persisted.LastCheckedAt})
		return m.Status(), nil
	}
	asset, ok := m.findAsset(*rel)
	if !ok {
		err := errors.New("release does not contain a valid asset for this platform")
		m.setStatus(Status{State: Failed, CurrentVersion: m.info.Version, LastCheckedAt: m.persisted.LastCheckedAt, Message: "A nova versão ainda não está disponível para este computador."})
		return m.Status(), err
	}
	m.mu.Lock()
	m.pending = &releaseAsset{Version: normalizeVersion(rel.TagName), Notes: rel.Body, Date: rel.PublishedAt, Asset: asset}
	m.mu.Unlock()
	m.setStatus(Status{State: Available, CurrentVersion: m.info.Version, AvailableVersion: normalizeVersion(rel.TagName), ReleaseNotes: rel.Body, PublishedAt: rel.PublishedAt, LastCheckedAt: m.persisted.LastCheckedAt, TotalBytes: asset.Size})
	return m.Status(), nil
}

func (m *Manager) latest(ctx context.Context) (*release, string, bool, error) {
	repo := strings.Trim(strings.TrimSpace(m.info.Repository), "/")
	if len(strings.Split(repo, "/")) != 2 {
		return nil, "", false, errors.New("invalid release repository")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(m.options.APIBaseURL, "/")+"/repos/"+repo+"/releases/latest", nil)
	if err != nil {
		return nil, "", false, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "cash-updater")
	m.mu.RLock()
	if m.persisted.ETag != "" {
		req.Header.Set("If-None-Match", m.persisted.ETag)
	}
	m.mu.RUnlock()
	resp, err := m.options.HTTPClient.Do(req)
	if err != nil {
		return nil, "", false, err
	}
	defer resp.Body.Close()
	if resp.Request.URL.Scheme != "https" && !m.options.AllowHTTP {
		return nil, "", false, errors.New("release request was redirected to an insecure URL")
	}
	if resp.StatusCode == http.StatusNotModified {
		return nil, resp.Header.Get("ETag"), true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, resp.Header.Get("ETag"), false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, resp.Header.Get("ETag"), false, fmt.Errorf("release request returned %s", resp.Status)
	}
	var rel release
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2*1024*1024)).Decode(&rel); err != nil {
		return nil, resp.Header.Get("ETag"), false, fmt.Errorf("decode release: %w", err)
	}
	return &rel, resp.Header.Get("ETag"), false, nil
}

func (m *Manager) findAsset(rel release) (asset, bool) {
	name := fmt.Sprintf("cash_%s_%s_amd64", normalizeVersion(rel.TagName), m.options.GOOS)
	if m.options.GOOS == "windows" {
		name += "_setup.exe"
	} else {
		name += ".deb"
	}
	for _, item := range rel.Assets {
		if item.Name != name || item.Size <= 0 || item.Size > maxDownload || !validDigest(item.Digest) || !m.validAssetURL(item.BrowserDownloadURL) {
			continue
		}
		return item, true
	}
	return asset{}, false
}

func (m *Manager) validAssetURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Host != "" && (m.options.AllowHTTP || u.Scheme == "https")
}

// Install downloads the selected asset, verifies it, then starts the temporary
// helper. The desktop façade closes the Wails process after this method returns.
func (m *Manager) Install(ctx context.Context) (Status, error) {
	m.mu.RLock()
	pending := m.pending
	m.mu.RUnlock()
	if pending == nil {
		return m.Status(), errors.New("no update is available")
	}
	if strings.TrimSpace(m.options.HelperPath) == "" {
		m.setStatus(Status{State: Failed, CurrentVersion: m.info.Version, AvailableVersion: pending.Version, Message: "O atualizador auxiliar não está disponível nesta instalação."})
		return m.Status(), errors.New("updater helper path is empty")
	}
	path, err := m.download(ctx, *pending)
	if err != nil {
		m.setStatus(Status{State: Failed, CurrentVersion: m.info.Version, AvailableVersion: pending.Version, Message: "Não foi possível baixar a atualização com segurança. Tente novamente."})
		return m.Status(), err
	}
	m.setStatus(Status{State: Installing, CurrentVersion: m.info.Version, AvailableVersion: pending.Version, TotalBytes: pending.Asset.Size, DownloadedBytes: pending.Asset.Size})
	if err := m.startHelper(path); err != nil {
		m.setStatus(Status{State: Failed, CurrentVersion: m.info.Version, AvailableVersion: pending.Version, Message: "Não foi possível iniciar a instalação da atualização."})
		return m.Status(), err
	}
	return m.Status(), nil
}

func (m *Manager) download(ctx context.Context, pending releaseAsset) (string, error) {
	m.setStatus(Status{State: Downloading, CurrentVersion: m.info.Version, AvailableVersion: pending.Version, TotalBytes: pending.Asset.Size})
	if err := os.MkdirAll(m.downloadDir, 0o700); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pending.Asset.BrowserDownloadURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := m.options.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.Request.URL.Scheme != "https" && !m.options.AllowHTTP {
		return "", errors.New("update download was redirected to an insecure URL")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned %s", resp.Status)
	}
	if resp.ContentLength > maxDownload || (resp.ContentLength >= 0 && resp.ContentLength != pending.Asset.Size) {
		return "", errors.New("unexpected update size")
	}
	tmp, err := os.CreateTemp(m.downloadDir, ".cash-update-*")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	hash := sha256.New()
	reader := io.LimitReader(resp.Body, maxDownload+1)
	buf := make([]byte, 128*1024)
	var copied int64
	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			if _, err := hash.Write(buf[:n]); err != nil {
				_ = tmp.Close()
				return "", err
			}
			if _, err := tmp.Write(buf[:n]); err != nil {
				_ = tmp.Close()
				return "", err
			}
			copied += int64(n)
			m.setStatus(Status{State: Downloading, CurrentVersion: m.info.Version, AvailableVersion: pending.Version, TotalBytes: pending.Asset.Size, DownloadedBytes: copied})
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = tmp.Close()
			return "", readErr
		}
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if copied != pending.Asset.Size || copied > maxDownload {
		return "", errors.New("incomplete update download")
	}
	if !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), strings.TrimPrefix(pending.Asset.Digest, "sha256:")) {
		return "", errors.New("update digest mismatch")
	}
	final := filepath.Join(m.downloadDir, pending.Asset.Name)
	if err := os.Rename(tmpName, final); err != nil {
		return "", err
	}
	return final, nil
}

func (m *Manager) startHelper(packagePath string) error {
	data, err := os.ReadFile(m.options.HelperPath)
	if err != nil {
		return fmt.Errorf("read updater helper: %w", err)
	}
	if err := os.MkdirAll(m.downloadDir, 0o700); err != nil {
		return err
	}
	ext := filepath.Ext(m.options.HelperPath)
	helper, err := os.CreateTemp(m.downloadDir, "cash-updater-*"+ext)
	if err != nil {
		return err
	}
	helperPath := helper.Name()
	if _, err := helper.Write(data); err != nil {
		_ = helper.Close()
		return err
	}
	if err := helper.Close(); err != nil {
		return err
	}
	if err := os.Chmod(helperPath, 0o700); err != nil {
		return err
	}
	cmd := newCommand(helperPath,
		"--parent-pid", strconv.Itoa(os.Getpid()),
		"--package", packagePath,
		"--app", m.options.Executable,
		"--result", m.resultPath,
		"--platform", m.options.GOOS,
	)
	return cmd.Start()
}

func (m *Manager) setStatus(status Status) {
	m.mu.Lock()
	m.status = status
	listeners := append([]func(Status){}, m.listeners...)
	m.mu.Unlock()
	for _, listener := range listeners {
		listener(status)
	}
}

func (m *Manager) recordCheck(etag string) {
	m.mu.Lock()
	m.persisted.LastCheckedAt = m.options.Now().UTC().Format(time.RFC3339)
	if etag != "" {
		m.persisted.ETag = etag
	}
	m.status.LastCheckedAt = m.persisted.LastCheckedAt
	state := m.persisted
	m.mu.Unlock()
	_ = writeJSON(m.statePath, state)
}

func (m *Manager) loadPersisted() {
	var state persistedState
	if readJSON(m.statePath, &state) == nil {
		m.persisted = state
		m.status.LastCheckedAt = state.LastCheckedAt
	}
}

func (m *Manager) loadResult() {
	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if readJSON(m.resultPath, &result) != nil {
		return
	}
	_ = os.Remove(m.resultPath)
	if result.Success {
		m.status.Message = "Atualização concluída."
	} else if result.Message != "" {
		m.status.Message = result.Message
	}
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".cash-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func readJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}
func parseTime(value string) time.Time     { at, _ := time.Parse(time.RFC3339, value); return at }
func normalizeVersion(value string) string { return strings.TrimPrefix(value, "v") }
func validVersion(value string) bool       { return versionPattern.MatchString(value) }
func validDigest(value string) bool {
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return strings.HasPrefix(value, "sha256:") && len(strings.TrimPrefix(value, "sha256:")) == 64 && err == nil
}

func compareVersions(left, right string) int {
	a, b := versionPattern.FindStringSubmatch(left), versionPattern.FindStringSubmatch(right)
	if a == nil || b == nil {
		return 0
	}
	for i := 1; i <= 3; i++ {
		x, _ := strconv.Atoi(a[i])
		y, _ := strconv.Atoi(b[i])
		if x < y {
			return -1
		}
		if x > y {
			return 1
		}
	}
	return 0
}
