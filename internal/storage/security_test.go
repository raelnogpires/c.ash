package storage

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"c.ash/internal/domain"
)

const testPassword = "senha-de-teste-segura"

func TestEncryptionLifecycle(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "cash.db")
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	account := domain.Account{ID: "a", Name: "Principal", Type: domain.AccountChecking, OpeningDate: "2026-08-01", CreatedAt: "2026-08-01T00:00:00Z"}
	if err := store.InsertAccount(ctx, account, account.CreatedAt); err != nil {
		t.Fatal(err)
	}
	result, err := store.EnableEncryption(ctx, testPassword, testPassword, "test")
	if err != nil {
		t.Fatal(err)
	}
	if result.RecoveryKey == "" || !result.Status.Enabled {
		t.Fatalf("result=%+v", result)
	}
	store.db.SetConnMaxLifetime(time.Nanosecond)
	time.Sleep(time.Millisecond)
	if accounts, err := store.Accounts(ctx); err != nil || len(accounts) != 1 {
		t.Fatalf("reopen encrypted pooled connection: accounts=%+v err=%v", accounts, err)
	}
	store.db.SetConnMaxLifetime(0)
	header, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.HasPrefix(header, []byte("SQLite format 3\x00")) {
		t.Fatal("encrypted database retained the plaintext SQLite header")
	}
	keys, err := os.ReadFile(filepath.Join(filepath.Dir(path), "cash.keys"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(keys, []byte(testPassword)) || bytes.Contains(keys, []byte(result.RecoveryKey)) {
		t.Fatal("credential leaked into key metadata")
	}
	oldEnvelopeBackup := filepath.Join(filepath.Dir(path), "before-password-change.cashbackup")
	if _, err := store.CreateBackupAt(ctx, oldEnvelopeBackup, BackupKindManual, "test"); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store, err = Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if status := store.SecurityStatus(); !status.Enabled || !status.Locked {
		t.Fatalf("status=%+v", status)
	}
	if _, err := store.Accounts(ctx); !errors.Is(err, ErrLocked) {
		t.Fatalf("locked read error=%v", err)
	}
	if err := store.Unlock(ctx, "senha-incorreta", ""); !errors.Is(err, ErrInvalidCredential) {
		t.Fatalf("wrong password error=%v", err)
	}
	if err := store.Unlock(ctx, testPassword, ""); err != nil {
		t.Fatal(err)
	}
	if accounts, err := store.Accounts(ctx); err != nil || len(accounts) != 1 || accounts[0].Name != account.Name {
		t.Fatalf("accounts=%+v err=%v", accounts, err)
	}
	newPassword := "uma-nova-senha-segura"
	if err := store.ChangePassword(ctx, testPassword, newPassword, newPassword, "test"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RestoreBackup(ctx, oldEnvelopeBackup, RestoreCredential{}, "test"); err != nil {
		t.Fatal(err)
	}
	if err := store.Lock(); err != nil {
		t.Fatal(err)
	}
	if err := store.Unlock(ctx, testPassword, ""); !errors.Is(err, ErrInvalidCredential) {
		t.Fatalf("old password error=%v", err)
	}
	if err := store.Unlock(ctx, newPassword, ""); err != nil {
		t.Fatalf("new password after restore: %v", err)
	}
	if err := store.Lock(); err != nil {
		t.Fatal(err)
	}
	if err := store.RecoverPassword(ctx, result.RecoveryKey, testPassword, testPassword, "test"); err != nil {
		t.Fatal(err)
	}
	if err := store.DisableEncryption(ctx, testPassword, "", "test"); err != nil {
		t.Fatal(err)
	}
	header, _ = os.ReadFile(path)
	if !bytes.HasPrefix(header, []byte("SQLite format 3\x00")) {
		t.Fatal("decrypted database does not have a SQLite header")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), "cash.keys")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("key metadata still exists: %v", err)
	}
	accounts, err := store.Accounts(ctx)
	if err != nil || len(accounts) != 1 {
		t.Fatalf("accounts after disable=%+v err=%v", accounts, err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestBackupExportAndRestoreRoundTrip(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := Open(ctx, filepath.Join(dir, "cash.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	account := domain.Account{ID: "a", Name: "Conta; Açúcar", Type: domain.AccountChecking, OpeningDate: "2026-08-01", CreatedAt: "2026-08-01T00:00:00Z"}
	if err := store.InsertAccount(ctx, account, account.CreatedAt); err != nil {
		t.Fatal(err)
	}
	active := domain.Transaction{ID: "active", Kind: domain.Income, AmountCents: 12345, AccountID: account.ID, Description: "Linha 1; \"cotação\"\nLinha 2", OccurrenceDate: "2026-08-02", CreatedAt: "2026-08-02T10:00:00Z", UpdatedAt: "2026-08-02T10:00:00Z"}
	trashed := domain.Transaction{ID: "trashed", Kind: domain.Expense, AmountCents: 100, AccountID: account.ID, CategoryID: "food", Description: "Removida", OccurrenceDate: "2026-08-03", CreatedAt: "2026-08-03T10:00:00Z", UpdatedAt: "2026-08-03T10:00:00Z"}
	for _, transaction := range []domain.Transaction{active, trashed} {
		if err := store.InsertTransaction(ctx, transaction, transaction.CreatedAt); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.SetTransactionDeletedAt(ctx, trashed.ID, "2026-08-04T00:00:00Z", "2026-08-04T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	backupPath := filepath.Join(dir, "portable.cashbackup")
	backup, err := store.CreateBackupAt(ctx, backupPath, BackupKindManual, "test")
	if err != nil {
		t.Fatal(err)
	}
	if backup.Manifest.SchemaVersion != latestSchemaVersion() || backup.Manifest.Encrypted {
		t.Fatalf("manifest=%+v", backup.Manifest)
	}
	archive, err := zip.OpenReader(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(archive.File) != 2 {
		t.Fatalf("archive files=%d", len(archive.File))
	}
	archive.Close()
	csvPath := filepath.Join(dir, "export.csv")
	if err := store.ExportData(ctx, ExportCSV, csvPath, "test"); err != nil {
		t.Fatal(err)
	}
	csvData, _ := os.ReadFile(csvPath)
	if !bytes.HasPrefix(csvData, []byte{0xef, 0xbb, 0xbf}) || !bytes.Contains(csvData, []byte("Conta; Açúcar")) || bytes.Contains(csvData, []byte("Removida")) || !bytes.Contains(csvData, []byte("123,45")) {
		t.Fatalf("unexpected CSV: %q", csvData)
	}
	jsonPath := filepath.Join(dir, "export.json")
	if err := store.ExportData(ctx, ExportJSON, jsonPath, "test"); err != nil {
		t.Fatal(err)
	}
	var exported map[string]any
	jsonData, _ := os.ReadFile(jsonPath)
	if err := json.Unmarshal(jsonData, &exported); err != nil {
		t.Fatal(err)
	}
	activeTransactions := exported["activeTransactions"].([]any)
	trashedTransactions := exported["trashedTransactions"].([]any)
	if len(activeTransactions) != 1 || len(trashedTransactions) != 1 || strings.Contains(string(jsonData), "cash.keys") || strings.Contains(string(jsonData), dir) {
		t.Fatalf("unexpected JSON export: %s", jsonData)
	}
	if err := store.DeleteTransaction(ctx, active.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.RestoreBackup(ctx, backupPath, RestoreCredential{}, "test"); err != nil {
		t.Fatal(err)
	}
	if transactions, err := store.Transactions(ctx); err != nil || len(transactions) != 1 || transactions[0].ID != active.ID {
		t.Fatalf("restored=%+v err=%v", transactions, err)
	}
}

func TestAutomaticBackupScheduleAndRetention(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "cash.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	status, err := store.RunAutomaticBackup(ctx, "test", now)
	if err != nil || status.AutomaticDue {
		t.Fatalf("status=%+v err=%v", status, err)
	}
	if _, err := store.CreateBackup(ctx, BackupKindManual, "test"); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < automaticRetention+3; index++ {
		preferences, _ := store.readBackupPreferences()
		preferences.LastAutomaticAt = ""
		if err := store.writeBackupPreferences(preferences); err != nil {
			t.Fatal(err)
		}
		if _, err := store.RunAutomaticBackup(ctx, "test", now.Add(time.Duration(index+1)*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	automatic, _ := filepath.Glob(filepath.Join(store.defaultBackupFolder(), "cash-automatic-*.cashbackup"))
	manual, _ := filepath.Glob(filepath.Join(store.defaultBackupFolder(), "cash-manual-*.cashbackup"))
	if len(automatic) != automaticRetention || len(manual) != 1 {
		t.Fatalf("automatic=%d manual=%d", len(automatic), len(manual))
	}
}

func TestEncryptedBackupRestoresAcrossInstallations(t *testing.T) {
	ctx := context.Background()
	sourceDir := t.TempDir()
	source, err := Open(ctx, filepath.Join(sourceDir, "cash.db"))
	if err != nil {
		t.Fatal(err)
	}
	account := domain.Account{ID: "portable", Name: "Portátil", Type: domain.AccountChecking, OpeningDate: "2026-08-01", CreatedAt: "x"}
	if err := source.InsertAccount(ctx, account, "x"); err != nil {
		t.Fatal(err)
	}
	if _, err := source.EnableEncryption(ctx, testPassword, testPassword, "test"); err != nil {
		t.Fatal(err)
	}
	backupPath := filepath.Join(sourceDir, "encrypted.cashbackup")
	info, err := source.CreateBackupAt(ctx, backupPath, BackupKindManual, "test")
	if err != nil || !info.Manifest.Encrypted {
		t.Fatalf("backup=%+v err=%v", info, err)
	}
	if err := source.Close(); err != nil {
		t.Fatal(err)
	}
	destination, err := Open(ctx, filepath.Join(t.TempDir(), "cash.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer destination.Close()
	if _, err := destination.RestoreBackup(ctx, backupPath, RestoreCredential{Password: "wrong-password"}, "test"); !errors.Is(err, ErrInvalidCredential) {
		t.Fatalf("wrong credential error=%v", err)
	}
	if _, err := destination.RestoreBackup(ctx, backupPath, RestoreCredential{Password: testPassword}, "test"); err != nil {
		t.Fatal(err)
	}
	if status := destination.SecurityStatus(); !status.Enabled || status.Locked {
		t.Fatalf("status=%+v", status)
	}
	accounts, err := destination.Accounts(ctx)
	if err != nil || len(accounts) != 1 || accounts[0].ID != account.ID {
		t.Fatalf("accounts=%+v err=%v", accounts, err)
	}
}

func TestBackupRejectsTamperingAndOperationJournalRollsBack(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "cash.db")
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	backupPath := filepath.Join(dir, "valid.cashbackup")
	if _, err := store.CreateBackupAt(ctx, backupPath, BackupKindManual, "test"); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	archive, err := zip.OpenReader(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	tamperedPath := filepath.Join(dir, "tampered.cashbackup")
	tampered, err := os.Create(tamperedPath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(tampered)
	for _, file := range archive.File {
		data, readErr := readZipFile(file)
		if readErr != nil {
			t.Fatal(readErr)
		}
		if file.Name == "cash.db" {
			data[0] ^= 0xff
		}
		entry, createErr := writer.Create(file.Name)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, err := entry.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	archive.Close()
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := tampered.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := inspectBackupArchive(tamperedPath, ""); !errors.Is(err, ErrInvalidBackup) {
		t.Fatalf("tampered archive error=%v", err)
	}
	rollback := path + ".rollback"
	if err := os.Rename(path, rollback); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("interrupted replacement"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeJSONAtomic(path+".operation.json", operationJournal{Operation: "restore", DatabaseBackup: rollback}, 0o600); err != nil {
		t.Fatal(err)
	}
	recovered, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer recovered.Close()
	if err := integrityCheck(ctx, recovered.db); err != nil {
		t.Fatalf("recovered database: %v", err)
	}
}
