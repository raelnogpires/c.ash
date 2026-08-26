package storage

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	BackupFormatVersion    = 1
	BackupKindManual       = "manual"
	BackupKindAutomatic    = "automatic"
	BackupKindSafety       = "safety"
	BackupKindPreRestore   = "pre-restore"
	BackupKindPreMigration = "pre-migration"
	BackupKindPreSecurity  = "pre-security"
	automaticInterval      = 7 * 24 * time.Hour
	automaticRetention     = 12
)

var (
	ErrInvalidBackup     = errors.New("invalid backup archive")
	ErrUnsupportedBackup = errors.New("unsupported backup archive")
	ErrNewerSchema       = errors.New("backup uses a newer schema")
)

type BackupManifest struct {
	FormatVersion  int    `json:"formatVersion"`
	CreatedAt      string `json:"createdAt"`
	ApplicationVer string `json:"applicationVersion"`
	SchemaVersion  int    `json:"schemaVersion"`
	Encrypted      bool   `json:"encrypted"`
	PayloadSHA256  string `json:"payloadSha256"`
	Kind           string `json:"kind"`
}

type BackupInfo struct {
	Path     string         `json:"path"`
	Manifest BackupManifest `json:"manifest"`
}

type BackupStatus struct {
	Folder            string `json:"folder"`
	DefaultFolder     string `json:"defaultFolder"`
	LastAutomaticAt   string `json:"lastAutomaticAt,omitempty"`
	LastAutomaticPath string `json:"lastAutomaticPath,omitempty"`
	LastError         string `json:"lastError,omitempty"`
	NextDueAt         string `json:"nextDueAt,omitempty"`
	AutomaticDue      bool   `json:"automaticDue"`
}

type backupPreferences struct {
	Folder            string `json:"folder,omitempty"`
	LastAutomaticAt   string `json:"lastAutomaticAt,omitempty"`
	LastAutomaticPath string `json:"lastAutomaticPath,omitempty"`
	LastError         string `json:"lastError,omitempty"`
}

type RestoreCredential struct {
	Password    string `json:"password,omitempty"`
	RecoveryKey string `json:"recoveryKey,omitempty"`
}

type ExportFormat string

const (
	ExportCSV  ExportFormat = "csv"
	ExportJSON ExportFormat = "json"
)

type operationJournal struct {
	Operation      string `json:"operation"`
	DatabaseBackup string `json:"databaseBackup,omitempty"`
	KeysBackup     string `json:"keysBackup,omitempty"`
	HadKeys        bool   `json:"hadKeys"`
}

func (s *Store) backupPreferencesPath() string {
	return filepath.Join(filepath.Dir(s.path), "backup-settings.json")
}

func (s *Store) defaultBackupFolder() string {
	return filepath.Join(filepath.Dir(s.path), "backups")
}

func (s *Store) readBackupPreferences() (backupPreferences, error) {
	var preferences backupPreferences
	data, err := os.ReadFile(s.backupPreferencesPath())
	if errors.Is(err, os.ErrNotExist) {
		return preferences, nil
	}
	if err != nil {
		return preferences, fmt.Errorf("read backup preferences: %w", err)
	}
	if err := json.Unmarshal(data, &preferences); err != nil {
		return preferences, fmt.Errorf("decode backup preferences: %w", err)
	}
	return preferences, nil
}

func (s *Store) writeBackupPreferences(preferences backupPreferences) error {
	return writeJSONAtomic(s.backupPreferencesPath(), preferences, 0o600)
}

func (s *Store) BackupStatus(now time.Time) (BackupStatus, error) {
	preferences, err := s.readBackupPreferences()
	if err != nil {
		return BackupStatus{}, err
	}
	defaultFolder := s.defaultBackupFolder()
	folder := preferences.Folder
	if folder == "" {
		folder = defaultFolder
	}
	if err := os.MkdirAll(folder, 0o700); err != nil {
		return BackupStatus{}, fmt.Errorf("create backup folder: %w", err)
	}
	status := BackupStatus{Folder: folder, DefaultFolder: defaultFolder, LastAutomaticAt: preferences.LastAutomaticAt, LastAutomaticPath: preferences.LastAutomaticPath, LastError: preferences.LastError, AutomaticDue: true}
	if last, parseErr := time.Parse(time.RFC3339Nano, preferences.LastAutomaticAt); parseErr == nil {
		next := last.Add(automaticInterval)
		status.NextDueAt = next.Format(time.RFC3339Nano)
		status.AutomaticDue = !now.Before(next)
	}
	return status, nil
}

func (s *Store) SetBackupFolder(path string) error {
	path = filepath.Clean(strings.TrimSpace(path))
	if !filepath.IsAbs(path) {
		return errors.New("backup folder must be absolute")
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("create backup folder: %w", err)
	}
	preferences, err := s.readBackupPreferences()
	if err != nil {
		return err
	}
	preferences.Folder = path
	return s.writeBackupPreferences(preferences)
}

func (s *Store) ResetBackupFolder() error {
	preferences, err := s.readBackupPreferences()
	if err != nil {
		return err
	}
	preferences.Folder = ""
	return s.writeBackupPreferences(preferences)
}

func (s *Store) CreateBackup(ctx context.Context, kind, version string) (BackupInfo, error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	return s.createBackupLocked(ctx, kind, version)
}

func (s *Store) createBackupLocked(ctx context.Context, kind, version string) (BackupInfo, error) {
	return s.createBackupAtLocked(ctx, kind, version, "")
}

func (s *Store) CreateBackupAt(ctx context.Context, path, kind, version string) (BackupInfo, error) {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	return s.createBackupAtLocked(ctx, kind, version, path)
}

func (s *Store) createBackupAtLocked(ctx context.Context, kind, version, destination string) (BackupInfo, error) {
	if s.db == nil {
		return BackupInfo{}, ErrLocked
	}
	preferences, err := s.readBackupPreferences()
	if err != nil {
		return BackupInfo{}, err
	}
	folder := preferences.Folder
	if folder == "" {
		folder = s.defaultBackupFolder()
	}
	if err := os.MkdirAll(folder, 0o700); err != nil {
		return BackupInfo{}, fmt.Errorf("create backup folder: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return BackupInfo{}, fmt.Errorf("checkpoint database: %w", err)
	}
	payload, err := os.ReadFile(s.path)
	if err != nil {
		return BackupInfo{}, fmt.Errorf("read database snapshot: %w", err)
	}
	digest := sha256.Sum256(payload)
	schema, err := schemaVersion(ctx, s.db)
	if err != nil {
		return BackupInfo{}, err
	}
	now := time.Now().UTC()
	manifest := BackupManifest{FormatVersion: BackupFormatVersion, CreatedAt: now.Format(time.RFC3339Nano), ApplicationVer: version, SchemaVersion: schema, Encrypted: s.encrypted, PayloadSHA256: hex.EncodeToString(digest[:]), Kind: kind}
	name := fmt.Sprintf("cash-%s-%s.cashbackup", kind, now.Format("20060102-150405.000000000"))
	path := destination
	if path == "" {
		path = filepath.Join(folder, name)
	} else if !filepath.IsAbs(path) {
		return BackupInfo{}, errors.New("backup destination must be absolute")
	}
	if !strings.EqualFold(filepath.Ext(path), ".cashbackup") {
		path += ".cashbackup"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return BackupInfo{}, err
	}
	if err := writeBackupArchive(path, manifest, payload, s.keyPath); err != nil {
		return BackupInfo{}, err
	}
	info := BackupInfo{Path: path, Manifest: manifest}
	if kind == BackupKindAutomatic {
		preferences.LastAutomaticAt = manifest.CreatedAt
		preferences.LastAutomaticPath = path
		preferences.LastError = ""
		if err := s.writeBackupPreferences(preferences); err != nil {
			return BackupInfo{}, err
		}
		if err := pruneAutomaticBackups(folder); err != nil {
			return BackupInfo{}, err
		}
	}
	return info, nil
}

func (s *Store) RunAutomaticBackup(ctx context.Context, version string, now time.Time) (BackupStatus, error) {
	status, err := s.BackupStatus(now)
	if err != nil || !status.AutomaticDue {
		return status, err
	}
	_, backupErr := s.CreateBackup(ctx, BackupKindAutomatic, version)
	if backupErr != nil {
		preferences, readErr := s.readBackupPreferences()
		if readErr == nil {
			preferences.LastError = backupErr.Error()
			_ = s.writeBackupPreferences(preferences)
		}
	}
	updated, statusErr := s.BackupStatus(now)
	return updated, errors.Join(backupErr, statusErr)
}

func writeBackupArchive(path string, manifest BackupManifest, payload []byte, keysPath string) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".cashbackup-*.tmp")
	if err != nil {
		return fmt.Errorf("create backup archive: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	archive := zip.NewWriter(temporary)
	manifestData, _ := json.MarshalIndent(manifest, "", "  ")
	for _, item := range []struct {
		name string
		data []byte
	}{{"manifest.json", manifestData}, {"cash.db", payload}} {
		writer, createErr := archive.CreateHeader(&zip.FileHeader{Name: item.name, Method: zip.Deflate})
		if createErr != nil {
			_ = archive.Close()
			_ = temporary.Close()
			return createErr
		}
		if _, err = writer.Write(item.data); err != nil {
			_ = archive.Close()
			_ = temporary.Close()
			return err
		}
	}
	if manifest.Encrypted {
		keys, readErr := os.ReadFile(keysPath)
		if readErr != nil {
			return fmt.Errorf("read encryption metadata: %w", readErr)
		}
		writer, createErr := archive.CreateHeader(&zip.FileHeader{Name: "cash.keys", Method: zip.Deflate})
		if createErr != nil {
			return createErr
		}
		if _, err = writer.Write(keys); err != nil {
			return err
		}
	}
	if err := archive.Close(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("finish backup archive: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if _, err := inspectBackupArchive(temporaryPath, ""); err != nil {
		return fmt.Errorf("validate backup archive: %w", err)
	}
	if err := replaceAtomic(temporaryPath, path); err != nil {
		return fmt.Errorf("install backup archive: %w", err)
	}
	return syncDirectory(filepath.Dir(path))
}

func pruneAutomaticBackups(folder string) error {
	entries, err := filepath.Glob(filepath.Join(folder, "cash-automatic-*.cashbackup"))
	if err != nil {
		return err
	}
	sort.Strings(entries)
	for len(entries) > automaticRetention {
		if err := os.Remove(entries[0]); err != nil {
			return fmt.Errorf("prune automatic backup: %w", err)
		}
		entries = entries[1:]
	}
	return nil
}

func (s *Store) InspectBackup(path string) (BackupInfo, error) {
	manifest, err := inspectBackupArchive(path, "")
	if err != nil {
		return BackupInfo{}, err
	}
	return BackupInfo{Path: path, Manifest: manifest}, nil
}

func inspectBackupArchive(path, extractDir string) (BackupManifest, error) {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return BackupManifest{}, fmt.Errorf("%w: cannot open archive", ErrInvalidBackup)
	}
	defer archive.Close()
	files := map[string]*zip.File{}
	for _, file := range archive.File {
		if file.Name != "manifest.json" && file.Name != "cash.db" && file.Name != "cash.keys" {
			return BackupManifest{}, fmt.Errorf("%w: unexpected file", ErrInvalidBackup)
		}
		if files[file.Name] != nil || file.UncompressedSize64 > 2<<30 {
			return BackupManifest{}, fmt.Errorf("%w: invalid archive entry", ErrInvalidBackup)
		}
		files[file.Name] = file
	}
	if files["manifest.json"] == nil || files["cash.db"] == nil {
		return BackupManifest{}, fmt.Errorf("%w: required file missing", ErrInvalidBackup)
	}
	manifestBytes, err := readZipFile(files["manifest.json"])
	if err != nil {
		return BackupManifest{}, fmt.Errorf("%w: manifest unreadable", ErrInvalidBackup)
	}
	var manifest BackupManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return BackupManifest{}, fmt.Errorf("%w: manifest malformed", ErrInvalidBackup)
	}
	if manifest.FormatVersion != BackupFormatVersion {
		return BackupManifest{}, ErrUnsupportedBackup
	}
	if manifest.SchemaVersion > latestSchemaVersion() {
		return BackupManifest{}, ErrNewerSchema
	}
	if manifest.Encrypted != (files["cash.keys"] != nil) {
		return BackupManifest{}, fmt.Errorf("%w: encryption metadata mismatch", ErrInvalidBackup)
	}
	payload, err := readZipFile(files["cash.db"])
	if err != nil {
		return BackupManifest{}, fmt.Errorf("%w: database unreadable", ErrInvalidBackup)
	}
	digest := sha256.Sum256(payload)
	if !strings.EqualFold(manifest.PayloadSHA256, hex.EncodeToString(digest[:])) {
		return BackupManifest{}, fmt.Errorf("%w: payload digest mismatch", ErrInvalidBackup)
	}
	if extractDir != "" {
		if err := os.WriteFile(filepath.Join(extractDir, "cash.db"), payload, 0o600); err != nil {
			return BackupManifest{}, err
		}
		if files["cash.keys"] != nil {
			keys, readErr := readZipFile(files["cash.keys"])
			if readErr != nil {
				return BackupManifest{}, readErr
			}
			if err := os.WriteFile(filepath.Join(extractDir, "cash.keys"), keys, 0o600); err != nil {
				return BackupManifest{}, err
			}
		}
	}
	return manifest, nil
}

func readZipFile(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(io.LimitReader(reader, int64(file.UncompressedSize64)+1))
}

func (s *Store) RestoreBackup(ctx context.Context, path string, credential RestoreCredential, version string) (BackupInfo, error) {
	temporaryDir, err := os.MkdirTemp(filepath.Dir(s.path), ".restore-*")
	if err != nil {
		return BackupInfo{}, err
	}
	defer os.RemoveAll(temporaryDir)
	manifest, err := inspectBackupArchive(path, temporaryDir)
	if err != nil {
		return BackupInfo{}, err
	}
	var candidateKey []byte
	if manifest.Encrypted {
		candidateKeys := filepath.Join(temporaryDir, "cash.keys")
		if len(s.key) == 32 && validateDatabase(ctx, filepath.Join(temporaryDir, "cash.db"), s.key) == nil {
			candidateKey = append([]byte(nil), s.key...)
			// A backup from this database can carry an older password envelope.
			// Preserve the currently authenticated envelope when the raw database
			// key is compatible, so the current password remains valid after restore.
			if err := copyFileAtomic(s.keyPath, candidateKeys, 0o600); err != nil {
				zero(candidateKey)
				return BackupInfo{}, fmt.Errorf("preserve current encryption credentials: %w", err)
			}
		} else {
			candidateKey, err = unlockKeyFile(candidateKeys, credential.Password, credential.RecoveryKey)
			if err != nil {
				return BackupInfo{}, err
			}
		}
		defer zero(candidateKey)
	}
	if err := validateDatabase(ctx, filepath.Join(temporaryDir, "cash.db"), candidateKey); err != nil {
		return BackupInfo{}, fmt.Errorf("%w: database validation failed", ErrInvalidBackup)
	}
	if _, err := s.CreateBackup(ctx, BackupKindPreRestore, version); err != nil {
		return BackupInfo{}, fmt.Errorf("create pre-restore backup: %w", err)
	}
	s.opMu.Lock()
	defer s.opMu.Unlock()
	if err := s.installDatabaseLocked(ctx, "restore", filepath.Join(temporaryDir, "cash.db"), filepath.Join(temporaryDir, "cash.keys"), manifest.Encrypted, candidateKey); err != nil {
		return BackupInfo{}, err
	}
	return BackupInfo{Path: path, Manifest: manifest}, nil
}

func (s *Store) installDatabaseLocked(ctx context.Context, operation, candidateDB, candidateKeys string, encrypted bool, key []byte) error {
	if s.db != nil {
		_, _ = s.db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`)
		if err := s.db.Close(); err != nil {
			return err
		}
		s.db = nil
	}
	dbRollback := s.path + ".rollback"
	keysRollback := s.keyPath + ".rollback"
	_ = os.Remove(dbRollback)
	_ = os.Remove(keysRollback)
	journal := operationJournal{Operation: operation, DatabaseBackup: dbRollback, KeysBackup: keysRollback, HadKeys: fileExists(s.keyPath)}
	if err := writeJSONAtomic(s.path+".operation.json", journal, 0o600); err != nil {
		return err
	}
	if fileExists(s.path) {
		if err := os.Rename(s.path, dbRollback); err != nil {
			return err
		}
	}
	if journal.HadKeys {
		if err := os.Rename(s.keyPath, keysRollback); err != nil {
			_ = os.Rename(dbRollback, s.path)
			return err
		}
	}
	installErr := copyFileAtomic(candidateDB, s.path, 0o600)
	if installErr == nil && encrypted {
		installErr = copyFileAtomic(candidateKeys, s.keyPath, 0o600)
	}
	if installErr == nil && !encrypted {
		_ = os.Remove(s.keyPath)
	}
	var db *sql.DB
	installedKey := append([]byte(nil), key...)
	if installErr == nil {
		db, _, installErr = openSQLite(s.path, installedKey)
	}
	if installErr == nil {
		installErr = integrityCheck(ctx, db)
	}
	if installErr == nil {
		installErr = ApplyMigrations(ctx, db, migrationFiles)
	}
	if installErr != nil {
		if db != nil {
			_ = db.Close()
		}
		zero(installedKey)
		_ = os.Remove(s.path)
		_ = os.Remove(s.keyPath)
		_ = os.Rename(dbRollback, s.path)
		if journal.HadKeys {
			_ = os.Rename(keysRollback, s.keyPath)
		}
		oldKey := append([]byte(nil), s.key...)
		reopened, _, reopenErr := openSQLite(s.path, oldKey)
		if reopenErr == nil {
			zero(s.key)
			s.key = oldKey
			s.db = reopened
		} else {
			zero(oldKey)
		}
		_ = os.Remove(s.path + ".operation.json")
		return fmt.Errorf("install database: %w", installErr)
	}
	zero(s.key)
	s.key = installedKey
	s.encrypted = encrypted
	s.db = db
	_ = os.Remove(dbRollback)
	_ = os.Remove(keysRollback)
	_ = os.Remove(s.path + ".operation.json")
	return syncDirectory(filepath.Dir(s.path))
}

func (s *Store) recoverOperation() error {
	path := s.path + ".operation.json"
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read operation journal: %w", err)
	}
	var journal operationJournal
	if err := json.Unmarshal(data, &journal); err != nil {
		return fmt.Errorf("decode operation journal: %w", err)
	}
	if fileExists(journal.DatabaseBackup) {
		_ = os.Remove(s.path)
		if err := os.Rename(journal.DatabaseBackup, s.path); err != nil {
			return fmt.Errorf("recover database operation: %w", err)
		}
		_ = os.Remove(s.keyPath)
		if journal.HadKeys && fileExists(journal.KeysBackup) {
			if err := os.Rename(journal.KeysBackup, s.keyPath); err != nil {
				return fmt.Errorf("recover encryption metadata: %w", err)
			}
		}
	}
	return os.Remove(path)
}

func validateDatabase(ctx context.Context, path string, key []byte) error {
	db, _, err := openSQLite(path, key)
	if err != nil {
		return err
	}
	defer db.Close()
	return integrityCheck(ctx, db)
}

func integrityCheck(ctx context.Context, db *sql.DB) error {
	var result string
	if err := db.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&result); err != nil {
		return err
	}
	if result != "ok" {
		return fmt.Errorf("integrity check: %s", result)
	}
	rows, err := db.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		return errors.New("foreign key check failed")
	}
	return rows.Err()
}

func (s *Store) ExportData(ctx context.Context, format ExportFormat, path, appVersion string) error {
	s.opMu.RLock()
	defer s.opMu.RUnlock()
	if s.db == nil {
		return ErrLocked
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var data []byte
	switch format {
	case ExportCSV:
		data, err = exportCSV(ctx, tx)
	case ExportJSON:
		data, err = exportJSON(ctx, tx, appVersion)
	default:
		err = errors.New("unsupported export format")
	}
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return writeFileAtomic(path, data, 0o600)
}

func exportCSV(ctx context.Context, tx *sql.Tx) ([]byte, error) {
	rows, err := tx.QueryContext(ctx, transactionSelect+` WHERE t.deleted_at IS NULL ORDER BY t.occurrence_date, t.created_at, t.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var output strings.Builder
	output.WriteRune('\ufeff')
	writer := csv.NewWriter(&output)
	writer.Comma = ';'
	header := []string{"id", "tipo", "valor_brl", "data_ocorrencia", "criado_em", "atualizado_em", "conta_origem_id", "conta_origem", "conta_destino_id", "conta_destino", "categoria_id", "categoria", "descricao", "importacao_automatica", "banco_importacao", "chave_importacao"}
	if err := writer.Write(header); err != nil {
		return nil, err
	}
	for rows.Next() {
		transaction, scanErr := scanTransaction(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		value := strconv.FormatInt(transaction.AmountCents/100, 10) + "," + fmt.Sprintf("%02d", transaction.AmountCents%100)
		record := []string{transaction.ID, string(transaction.Kind), value, transaction.OccurrenceDate, transaction.CreatedAt, transaction.UpdatedAt, transaction.AccountID, transaction.AccountName, transaction.DestinationAccountID, transaction.DestinationAccountName, transaction.CategoryID, transaction.CategoryName, transaction.Description, strconv.FormatBool(transaction.AutomaticImport), string(transaction.ImportBank), transaction.ImportKey}
		if err := writer.Write(record); err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	writer.Flush()
	return []byte(output.String()), writer.Error()
}

func exportJSON(ctx context.Context, tx *sql.Tx, appVersion string) ([]byte, error) {
	tables := []struct {
		key   string
		table string
		order string
		where string
	}{
		{"accounts", "accounts", "created_at, id", ""}, {"categories", "categories", "rowid", ""},
		{"activeTransactions", "transactions", "occurrence_date, created_at, id", "deleted_at IS NULL"},
		{"trashedTransactions", "transactions", "occurrence_date, created_at, id", "deleted_at IS NOT NULL"},
		{"transactionRevisions", "transaction_revisions", "id", ""},
		{"fixedExpenses", "fixed_expenses", "created_at, id", ""}, {"fixedExpenseOccurrences", "fixed_expense_occurrences", "reference_month, id", ""},
		{"invoices", "credit_card_invoices", "closing_date, id", ""}, {"installments", "credit_card_installments", "created_at, id", ""},
		{"payments", "credit_card_payments", "occurrence_date, created_at, id", ""},
	}
	schema, err := schemaVersion(ctx, tx)
	if err != nil {
		return nil, err
	}
	document := map[string]any{"formatVersion": 1, "exportedAt": time.Now().UTC().Format(time.RFC3339Nano), "applicationVersion": appVersion, "schemaVersion": schema}
	profiles, err := rowsAsMaps(ctx, tx, "profile", "id", "")
	if err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		document["profile"] = nil
	} else {
		document["profile"] = profiles[0]
	}
	for _, table := range tables {
		items, queryErr := rowsAsMaps(ctx, tx, table.table, table.order, table.where)
		if queryErr != nil {
			return nil, queryErr
		}
		document[table.key] = items
	}
	return json.MarshalIndent(document, "", "  ")
}

type queryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func schemaVersion(ctx context.Context, query queryRower) (int, error) {
	var version string
	if err := query.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), '0') FROM schema_migrations`).Scan(&version); err != nil {
		return 0, fmt.Errorf("read schema version: %w", err)
	}
	prefix := strings.SplitN(version, "_", 2)[0]
	parsed, err := strconv.Atoi(prefix)
	if err != nil {
		return 0, fmt.Errorf("parse schema version: %w", err)
	}
	return parsed, nil
}

func latestSchemaVersion() int {
	entries, _ := migrationFiles.ReadDir("migrations")
	return len(entries)
}

func rowsAsMaps(ctx context.Context, tx *sql.Tx, table, order, where string) ([]map[string]any, error) {
	query := `SELECT * FROM ` + table
	if where != "" {
		query += ` WHERE ` + where
	}
	rows, err := tx.QueryContext(ctx, query+` ORDER BY `+order)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	items := []map[string]any{}
	for rows.Next() {
		values := make([]any, len(columns))
		destinations := make([]any, len(columns))
		for index := range values {
			destinations[index] = &values[index]
		}
		if err := rows.Scan(destinations...); err != nil {
			return nil, err
		}
		item := make(map[string]any, len(columns))
		for index, column := range columns {
			if bytes, ok := values[index].([]byte); ok {
				item[column] = string(bytes)
			} else {
				item[column] = values[index]
			}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func writeJSONAtomic(path string, value any, mode os.FileMode) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(path, data, mode)
}

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".cash-write-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := replaceAtomic(temporaryPath, path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

func copyFileAtomic(source, destination string, mode os.FileMode) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return writeFileAtomic(destination, data, mode)
}
