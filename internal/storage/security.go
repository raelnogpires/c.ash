package storage

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
)

const (
	keyFileVersion   = 1
	argonMemoryKiB   = 64 * 1024
	argonIterations  = 3
	minimumPassword  = 12
	databaseKeyBytes = 32
)

var (
	ErrInvalidCredential  = errors.New("invalid encryption credential")
	ErrWeakPassword       = errors.New("encryption password must have at least 12 characters")
	ErrPasswordMismatch   = errors.New("password confirmation does not match")
	ErrEncryptionEnabled  = errors.New("database encryption is already enabled")
	ErrEncryptionDisabled = errors.New("database encryption is disabled")
	ErrCorruptKeyFile     = errors.New("encryption metadata is invalid")
)

type SecurityStatus struct {
	Enabled bool `json:"enabled"`
	Locked  bool `json:"locked"`
}

type EncryptionResult struct {
	Status      SecurityStatus `json:"status"`
	RecoveryKey string         `json:"recoveryKey,omitempty"`
}

type kdfParameters struct {
	Algorithm  string `json:"algorithm"`
	MemoryKiB  uint32 `json:"memoryKiB"`
	Iterations uint32 `json:"iterations"`
	Lanes      uint8  `json:"lanes"`
}

type keyEnvelope struct {
	Salt       string `json:"salt,omitempty"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

type keyFile struct {
	FormatVersion int           `json:"formatVersion"`
	KDF           kdfParameters `json:"kdf"`
	Password      keyEnvelope   `json:"password"`
	Recovery      keyEnvelope   `json:"recovery"`
	CreatedAt     string        `json:"createdAt"`
	UpdatedAt     string        `json:"updatedAt"`
}

func (s *Store) SecurityStatus() SecurityStatus {
	s.opMu.RLock()
	defer s.opMu.RUnlock()
	return SecurityStatus{Enabled: s.encrypted, Locked: s.db == nil}
}

func (s *Store) Unlock(ctx context.Context, password, recoveryKey string) error {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	if !s.encrypted {
		return ErrEncryptionDisabled
	}
	if s.db != nil {
		return nil
	}
	key, err := unlockKeyFile(s.keyPath, password, recoveryKey)
	if err != nil {
		return err
	}
	db, _, err := openSQLite(s.path, key)
	if err != nil {
		zero(key)
		return ErrInvalidCredential
	}
	if err := integrityCheck(ctx, db); err != nil {
		_ = db.Close()
		zero(key)
		return ErrInvalidCredential
	}
	s.db = db
	s.key = key
	pending, err := hasPendingMigrations(ctx, db)
	if err != nil {
		_ = db.Close()
		s.db = nil
		zero(s.key)
		s.key = nil
		return err
	}
	if pending {
		if _, err := s.createBackupLocked(ctx, BackupKindPreMigration, s.version); err != nil {
			_ = db.Close()
			s.db = nil
			zero(s.key)
			s.key = nil
			return fmt.Errorf("create pre-migration backup: %w", err)
		}
	}
	if err := ApplyMigrations(ctx, db, migrationFiles); err != nil {
		_ = db.Close()
		s.db = nil
		zero(s.key)
		s.key = nil
		return err
	}
	return nil
}

func (s *Store) Lock() error {
	s.opMu.Lock()
	defer s.opMu.Unlock()
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			return err
		}
		s.db = nil
	}
	zero(s.key)
	s.key = nil
	return nil
}

func (s *Store) EnableEncryption(ctx context.Context, password, confirmation, version string) (EncryptionResult, error) {
	if err := validateNewPassword(password, confirmation); err != nil {
		return EncryptionResult{}, err
	}
	if s.SecurityStatus().Enabled {
		return EncryptionResult{}, ErrEncryptionEnabled
	}
	if _, err := s.CreateBackup(ctx, BackupKindPreSecurity, version); err != nil {
		return EncryptionResult{}, fmt.Errorf("create encryption safety backup: %w", err)
	}
	databaseKey, err := randomBytes(databaseKeyBytes)
	if err != nil {
		return EncryptionResult{}, err
	}
	defer zero(databaseKey)
	recovery, err := randomBytes(databaseKeyBytes)
	if err != nil {
		return EncryptionResult{}, err
	}
	defer zero(recovery)
	metadata, err := newKeyFile(password, databaseKey, recovery, time.Now().UTC())
	if err != nil {
		return EncryptionResult{}, err
	}
	candidateDB, cleanupDB, err := temporaryPath(filepath.Dir(s.path), ".encrypt-*.db")
	if err != nil {
		return EncryptionResult{}, err
	}
	defer cleanupDB()
	candidateKeys, cleanupKeys, err := temporaryPath(filepath.Dir(s.path), ".encrypt-*.keys")
	if err != nil {
		return EncryptionResult{}, err
	}
	defer cleanupKeys()
	if err := writeJSONAtomic(candidateKeys, metadata, 0o600); err != nil {
		return EncryptionResult{}, err
	}
	s.opMu.Lock()
	defer s.opMu.Unlock()
	if s.db == nil {
		return EncryptionResult{}, ErrLocked
	}
	if _, err := s.db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return EncryptionResult{}, err
	}
	if err := s.db.Close(); err != nil {
		return EncryptionResult{}, err
	}
	s.db = nil
	if err := convertDatabase(ctx, s.path, nil, candidateDB, databaseKey); err != nil {
		s.db, _, _ = openSQLite(s.path, nil)
		return EncryptionResult{}, fmt.Errorf("encrypt database: %w", err)
	}
	if err := s.installDatabaseLocked(ctx, "encrypt", candidateDB, candidateKeys, true, databaseKey); err != nil {
		return EncryptionResult{}, err
	}
	return EncryptionResult{Status: SecurityStatus{Enabled: true, Locked: false}, RecoveryKey: formatRecoveryKey(recovery)}, nil
}

func (s *Store) ChangePassword(ctx context.Context, currentPassword, newPassword, confirmation, version string) error {
	if err := validateNewPassword(newPassword, confirmation); err != nil {
		return err
	}
	key, err := unlockKeyFile(s.keyPath, currentPassword, "")
	if err != nil {
		return err
	}
	defer zero(key)
	if !sameKey(key, s.key) {
		return ErrInvalidCredential
	}
	if _, err := s.CreateBackup(ctx, BackupKindPreSecurity, version); err != nil {
		return err
	}
	s.opMu.Lock()
	defer s.opMu.Unlock()
	metadata, err := readKeyFile(s.keyPath)
	if err != nil {
		return err
	}
	metadata.Password, err = passwordEnvelope(newPassword, s.key, metadata.KDF)
	if err != nil {
		return err
	}
	metadata.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return writeJSONAtomic(s.keyPath, metadata, 0o600)
}

func (s *Store) RecoverPassword(ctx context.Context, recoveryKey, newPassword, confirmation, version string) error {
	if err := validateNewPassword(newPassword, confirmation); err != nil {
		return err
	}
	key, err := unlockKeyFile(s.keyPath, "", recoveryKey)
	if err != nil {
		return err
	}
	defer zero(key)
	if s.db != nil && !sameKey(key, s.key) {
		return ErrInvalidCredential
	}
	if s.db == nil {
		s.opMu.Lock()
		installedKey := append([]byte(nil), key...)
		db, _, openErr := openSQLite(s.path, installedKey)
		if openErr == nil {
			openErr = integrityCheck(ctx, db)
		}
		credentialErr := openErr != nil
		if openErr == nil {
			s.db, s.key = db, installedKey
			pending, migrationErr := hasPendingMigrations(ctx, db)
			if migrationErr == nil && pending {
				_, migrationErr = s.createBackupLocked(ctx, BackupKindPreMigration, s.version)
				if migrationErr != nil {
					migrationErr = fmt.Errorf("create pre-migration backup: %w", migrationErr)
				}
			}
			if migrationErr == nil {
				migrationErr = ApplyMigrations(ctx, db, migrationFiles)
			}
			openErr = migrationErr
		}
		if openErr != nil {
			if db != nil {
				_ = db.Close()
			}
			s.db = nil
			zero(s.key)
			s.key = nil
			zero(installedKey)
		}
		s.opMu.Unlock()
		if openErr != nil {
			if credentialErr {
				return ErrInvalidCredential
			}
			return openErr
		}
	}
	if _, err := s.CreateBackup(ctx, BackupKindPreSecurity, version); err != nil {
		return err
	}
	s.opMu.Lock()
	defer s.opMu.Unlock()
	metadata, err := readKeyFile(s.keyPath)
	if err != nil {
		return err
	}
	metadata.Password, err = passwordEnvelope(newPassword, key, metadata.KDF)
	if err != nil {
		return err
	}
	metadata.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := writeJSONAtomic(s.keyPath, metadata, 0o600); err != nil {
		return err
	}
	return nil
}

func (s *Store) DisableEncryption(ctx context.Context, password, recoveryKey, version string) error {
	if !s.SecurityStatus().Enabled {
		return ErrEncryptionDisabled
	}
	key, err := unlockKeyFile(s.keyPath, password, recoveryKey)
	if err != nil {
		return err
	}
	defer zero(key)
	if !sameKey(key, s.key) {
		return ErrInvalidCredential
	}
	if _, err := s.CreateBackup(ctx, BackupKindPreSecurity, version); err != nil {
		return err
	}
	candidateDB, cleanup, err := temporaryPath(filepath.Dir(s.path), ".decrypt-*.db")
	if err != nil {
		return err
	}
	defer cleanup()
	s.opMu.Lock()
	defer s.opMu.Unlock()
	if s.db == nil {
		return ErrLocked
	}
	if _, err := s.db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
		return err
	}
	if err := s.db.Close(); err != nil {
		return err
	}
	s.db = nil
	if err := convertDatabase(ctx, s.path, key, candidateDB, nil); err != nil {
		s.db, _, _ = openSQLite(s.path, s.key)
		return fmt.Errorf("decrypt database: %w", err)
	}
	if err := s.installDatabaseLocked(ctx, "decrypt", candidateDB, "", false, nil); err != nil {
		return err
	}
	return nil
}

func validateNewPassword(password, confirmation string) error {
	if len([]rune(password)) < minimumPassword {
		return ErrWeakPassword
	}
	if password != confirmation {
		return ErrPasswordMismatch
	}
	return nil
}

func newKeyFile(password string, databaseKey, recovery []byte, now time.Time) (keyFile, error) {
	lanes := runtime.NumCPU()
	if lanes < 1 {
		lanes = 1
	}
	if lanes > 4 {
		lanes = 4
	}
	metadata := keyFile{FormatVersion: keyFileVersion, KDF: kdfParameters{Algorithm: "argon2id", MemoryKiB: argonMemoryKiB, Iterations: argonIterations, Lanes: uint8(lanes)}, CreatedAt: now.Format(time.RFC3339Nano), UpdatedAt: now.Format(time.RFC3339Nano)}
	var err error
	metadata.Password, err = passwordEnvelope(password, databaseKey, metadata.KDF)
	if err != nil {
		return keyFile{}, err
	}
	metadata.Recovery, err = recoveryEnvelope(recovery, databaseKey)
	return metadata, err
}

func passwordEnvelope(password string, databaseKey []byte, parameters kdfParameters) (keyEnvelope, error) {
	salt, err := randomBytes(16)
	if err != nil {
		return keyEnvelope{}, err
	}
	defer zero(salt)
	wrappingKey := argon2.IDKey([]byte(password), salt, parameters.Iterations, parameters.MemoryKiB, parameters.Lanes, 32)
	defer zero(wrappingKey)
	envelope, err := sealKey(wrappingKey, databaseKey)
	if err == nil {
		envelope.Salt = base64.RawStdEncoding.EncodeToString(salt)
	}
	return envelope, err
}

func recoveryEnvelope(recovery, databaseKey []byte) (keyEnvelope, error) {
	return sealKey(recovery, databaseKey)
}

func sealKey(wrappingKey, databaseKey []byte) (keyEnvelope, error) {
	block, err := aes.NewCipher(wrappingKey)
	if err != nil {
		return keyEnvelope{}, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return keyEnvelope{}, err
	}
	nonce, err := randomBytes(aead.NonceSize())
	if err != nil {
		return keyEnvelope{}, err
	}
	defer zero(nonce)
	ciphertext := aead.Seal(nil, nonce, databaseKey, []byte("c.ash/database-key/v1"))
	return keyEnvelope{Nonce: base64.RawStdEncoding.EncodeToString(nonce), Ciphertext: base64.RawStdEncoding.EncodeToString(ciphertext)}, nil
}

func openEnvelope(envelope keyEnvelope, wrappingKey []byte) ([]byte, error) {
	nonce, err := base64.RawStdEncoding.DecodeString(envelope.Nonce)
	if err != nil {
		return nil, ErrCorruptKeyFile
	}
	ciphertext, err := base64.RawStdEncoding.DecodeString(envelope.Ciphertext)
	if err != nil {
		return nil, ErrCorruptKeyFile
	}
	block, err := aes.NewCipher(wrappingKey)
	if err != nil {
		return nil, ErrCorruptKeyFile
	}
	aead, err := cipher.NewGCM(block)
	if err != nil || len(nonce) != aead.NonceSize() {
		return nil, ErrCorruptKeyFile
	}
	key, err := aead.Open(nil, nonce, ciphertext, []byte("c.ash/database-key/v1"))
	if err != nil || len(key) != databaseKeyBytes {
		zero(key)
		return nil, ErrInvalidCredential
	}
	return key, nil
}

func unlockKeyFile(path, password, recoveryKey string) ([]byte, error) {
	metadata, err := readKeyFile(path)
	if err != nil {
		return nil, err
	}
	if password != "" {
		salt, err := base64.RawStdEncoding.DecodeString(metadata.Password.Salt)
		if err != nil || len(salt) < 16 {
			return nil, ErrCorruptKeyFile
		}
		wrappingKey := argon2.IDKey([]byte(password), salt, metadata.KDF.Iterations, metadata.KDF.MemoryKiB, metadata.KDF.Lanes, 32)
		key, openErr := openEnvelope(metadata.Password, wrappingKey)
		zero(wrappingKey)
		zero(salt)
		return key, openErr
	}
	if recoveryKey != "" {
		recovery, err := parseRecoveryKey(recoveryKey)
		if err != nil {
			return nil, ErrInvalidCredential
		}
		defer zero(recovery)
		return openEnvelope(metadata.Recovery, recovery)
	}
	return nil, ErrInvalidCredential
}

func readKeyFile(path string) (keyFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return keyFile{}, fmt.Errorf("read encryption metadata: %w", err)
	}
	var metadata keyFile
	if err := json.Unmarshal(data, &metadata); err != nil {
		return keyFile{}, ErrCorruptKeyFile
	}
	if metadata.FormatVersion != keyFileVersion || metadata.KDF.Algorithm != "argon2id" || metadata.KDF.MemoryKiB != argonMemoryKiB || metadata.KDF.Iterations != argonIterations || metadata.KDF.Lanes < 1 || metadata.KDF.Lanes > 4 {
		return keyFile{}, ErrCorruptKeyFile
	}
	return metadata, nil
}

func formatRecoveryKey(key []byte) string {
	digest := sha256.Sum256(append([]byte("c.ash/recovery/v1"), key...))
	payload := append(append([]byte(nil), key...), digest[:4]...)
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(payload)
	groups := make([]string, 0, (len(encoded)+4)/5)
	for len(encoded) > 0 {
		length := 5
		if len(encoded) < length {
			length = len(encoded)
		}
		groups = append(groups, encoded[:length])
		encoded = encoded[length:]
	}
	zero(payload)
	return strings.Join(groups, "-")
}

func parseRecoveryKey(value string) ([]byte, error) {
	normalized := strings.NewReplacer("-", "", " ", "", "\n", "", "\t", "").Replace(strings.ToUpper(value))
	payload, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(normalized)
	if err != nil || len(payload) != databaseKeyBytes+4 {
		zero(payload)
		return nil, ErrInvalidCredential
	}
	key := append([]byte(nil), payload[:databaseKeyBytes]...)
	digest := sha256.Sum256(append([]byte("c.ash/recovery/v1"), key...))
	valid := subtle.ConstantTimeCompare(payload[databaseKeyBytes:], digest[:4]) == 1
	zero(payload)
	if !valid {
		zero(key)
		return nil, ErrInvalidCredential
	}
	return key, nil
}

func convertDatabase(ctx context.Context, source string, sourceKey []byte, destination string, destinationKey []byte) error {
	_ = os.Remove(destination)
	db, _, err := openSQLite(source, sourceKey)
	if err != nil {
		return err
	}
	defer db.Close()
	keyLiteral := ""
	if len(destinationKey) > 0 {
		keyLiteral = "x'" + hex.EncodeToString(destinationKey) + "'"
	}
	if _, err := db.ExecContext(ctx, `ATTACH DATABASE ? AS converted KEY ?`, destination, keyLiteral); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `SELECT sqlcipher_export('converted')`); err != nil {
		_, _ = db.ExecContext(ctx, `DETACH DATABASE converted`)
		return err
	}
	if _, err := db.ExecContext(ctx, `DETACH DATABASE converted`); err != nil {
		return err
	}
	return validateDatabase(ctx, destination, destinationKey)
}

func temporaryPath(directory, pattern string) (string, func(), error) {
	file, err := os.CreateTemp(directory, pattern)
	if err != nil {
		return "", func() {}, err
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		return "", func() {}, err
	}
	if err := os.Remove(path); err != nil {
		return "", func() {}, err
	}
	return path, func() { _ = os.Remove(path) }, nil
}

func randomBytes(length int) ([]byte, error) {
	value := make([]byte, length)
	if _, err := rand.Read(value); err != nil {
		zero(value)
		return nil, err
	}
	return value, nil
}

func sameKey(left, right []byte) bool {
	return len(left) == databaseKeyBytes && len(right) == databaseKeyBytes && subtle.ConstantTimeCompare(left, right) == 1
}

func zero(value []byte) {
	for index := range value {
		value[index] = 0
	}
}
