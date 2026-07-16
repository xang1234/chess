package backup

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"chess-trainer/internal/storage"
)

const manifestVersion = 1

var databaseNames = map[string]string{
	"user.sqlite":    "profile",
	"library.sqlite": "library_metadata",
}

type Manifest struct {
	Version   int               `json:"version"`
	CreatedAt int64             `json:"createdAt"`
	Files     map[string]string `json:"files"`
}

type Service struct {
	paths     storage.Paths
	closeLive func() error
	now       func() time.Time
}

func NewService(paths storage.Paths, closeLive func() error) *Service {
	return &Service{paths: paths, closeLive: closeLive, now: time.Now}
}

func (s *Service) Create(ctx context.Context, destination string, includeLibrary bool) error {
	if strings.TrimSpace(destination) == "" {
		return errors.New("backup destination is required")
	}
	if err := s.paths.Ensure(); err != nil {
		return err
	}
	if err := s.validateDestination(destination); err != nil {
		return err
	}
	temporary, err := os.MkdirTemp(s.paths.Root, ".backup-export-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)

	sources := map[string]string{"user.sqlite": s.paths.UserDB}
	if includeLibrary {
		sources["library.sqlite"] = s.paths.LibraryDB
	}
	manifest := Manifest{Version: manifestVersion, CreatedAt: s.now().Unix(), Files: map[string]string{}}
	for _, name := range sortedKeys(sources) {
		if err := ctx.Err(); err != nil {
			return err
		}
		snapshot := filepath.Join(temporary, name)
		if err := vacuumInto(ctx, sources[name], snapshot); err != nil {
			return fmt.Errorf("snapshot %s: %w", name, err)
		}
		digest, err := fileDigest(snapshot)
		if err != nil {
			return err
		}
		manifest.Files[name] = digest
	}
	return writeArchive(destination, temporary, manifest)
}

func (s *Service) validateDestination(destination string) error {
	canonicalDestination, err := canonicalPath(destination)
	if err != nil {
		return fmt.Errorf("resolve backup destination: %w", err)
	}
	destinationInfo, destinationStatErr := os.Stat(canonicalDestination)
	if destinationStatErr != nil && !errors.Is(destinationStatErr, os.ErrNotExist) {
		return fmt.Errorf("inspect backup destination: %w", destinationStatErr)
	}
	for _, managed := range []string{s.paths.UserDB, s.paths.LibraryDB, s.paths.PuzzlesDB} {
		canonicalManaged, err := canonicalPath(managed)
		if err != nil {
			return fmt.Errorf("resolve managed database path: %w", err)
		}
		sameFile := false
		if destinationStatErr == nil {
			managedInfo, err := os.Stat(canonicalManaged)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("inspect managed database: %w", err)
			}
			if err == nil {
				sameFile = os.SameFile(destinationInfo, managedInfo)
			}
		}
		if canonicalDestination == canonicalManaged || sameFile {
			return fmt.Errorf("backup destination %q resolves to managed database %q", destination, managed)
		}
	}
	return nil
}

func canonicalPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err == nil {
		return resolved, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(absolute))
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(absolute)), nil
}

func (s *Service) Restore(ctx context.Context, source string) error {
	if strings.TrimSpace(source) == "" {
		return errors.New("backup path is required")
	}
	if err := s.paths.Ensure(); err != nil {
		return err
	}
	temporary, err := os.MkdirTemp(s.paths.Root, ".backup-restore-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)
	manifest, err := extractAndValidate(ctx, source, temporary)
	if err != nil {
		return err
	}
	if s.closeLive == nil {
		return errors.New("database close callback is not configured")
	}
	if err := s.closeLive(); err != nil {
		return fmt.Errorf("close live databases: %w", err)
	}
	return s.replaceDatabases(temporary, manifest)
}

func vacuumInto(ctx context.Context, source, destination string) error {
	if _, err := os.Stat(source); err != nil {
		return err
	}
	db, err := storage.Open(source)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.ExecContext(ctx, `VACUUM INTO ?`, destination)
	return err
}

func writeArchive(destination, snapshots string, manifest Manifest) (resultErr error) {
	directory := filepath.Dir(destination)
	temporary, err := os.CreateTemp(directory, ".chess-trainer-backup-*.zip")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() {
		if resultErr != nil {
			temporary.Close()
			os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return err
	}
	writer := zip.NewWriter(temporary)
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := writeArchiveBytes(writer, "manifest.json", encoded); err != nil {
		return err
	}
	for _, name := range sortedKeys(manifest.Files) {
		if err := writeArchiveFile(writer, name, filepath.Join(snapshots, name)); err != nil {
			return err
		}
	}
	if err := writer.Close(); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, destination); err != nil {
		return err
	}
	return nil
}

func writeArchiveBytes(writer *zip.Writer, name string, contents []byte) error {
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	header.SetMode(0o600)
	header.SetModTime(time.Unix(0, 0))
	destination, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = destination.Write(contents)
	return err
}

func writeArchiveFile(writer *zip.Writer, name, source string) error {
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()
	header := &zip.FileHeader{Name: name, Method: zip.Deflate}
	header.SetMode(0o600)
	header.SetModTime(time.Unix(0, 0))
	destination, err := writer.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(destination, file)
	return err
}

func extractAndValidate(ctx context.Context, source, destination string) (Manifest, error) {
	reader, err := zip.OpenReader(source)
	if err != nil {
		return Manifest{}, fmt.Errorf("open backup: %w", err)
	}
	defer reader.Close()
	files := make(map[string]*zip.File, len(reader.File))
	for _, file := range reader.File {
		if file.Name != "manifest.json" {
			if _, allowed := databaseNames[file.Name]; !allowed {
				return Manifest{}, fmt.Errorf("backup contains unexpected file %q", file.Name)
			}
		}
		if _, duplicate := files[file.Name]; duplicate {
			return Manifest{}, fmt.Errorf("backup contains duplicate file %q", file.Name)
		}
		files[file.Name] = file
	}
	manifestFile := files["manifest.json"]
	if manifestFile == nil {
		return Manifest{}, errors.New("backup manifest is missing")
	}
	manifest, err := decodeManifest(manifestFile)
	if err != nil {
		return Manifest{}, err
	}
	if manifest.Version != manifestVersion {
		return Manifest{}, fmt.Errorf("unsupported backup version %d", manifest.Version)
	}
	if manifest.Files["user.sqlite"] == "" {
		return Manifest{}, errors.New("backup does not contain user.sqlite")
	}
	if len(files) != len(manifest.Files)+1 {
		return Manifest{}, errors.New("backup file list does not match its manifest")
	}
	for _, name := range sortedKeys(manifest.Files) {
		if err := ctx.Err(); err != nil {
			return Manifest{}, err
		}
		if _, allowed := databaseNames[name]; !allowed {
			return Manifest{}, fmt.Errorf("manifest contains unexpected file %q", name)
		}
		archiveFile := files[name]
		if archiveFile == nil {
			return Manifest{}, fmt.Errorf("manifest file %q is missing", name)
		}
		extracted := filepath.Join(destination, name)
		if err := extractFile(archiveFile, extracted); err != nil {
			return Manifest{}, err
		}
		digest, err := fileDigest(extracted)
		if err != nil {
			return Manifest{}, err
		}
		if digest != strings.ToLower(manifest.Files[name]) {
			return Manifest{}, fmt.Errorf("backup checksum mismatch for %s", name)
		}
		if err := validateDatabase(extracted, databaseNames[name]); err != nil {
			return Manifest{}, fmt.Errorf("validate %s: %w", name, err)
		}
	}
	return manifest, nil
}

func decodeManifest(file *zip.File) (Manifest, error) {
	opened, err := file.Open()
	if err != nil {
		return Manifest{}, err
	}
	defer opened.Close()
	var manifest Manifest
	decoder := json.NewDecoder(io.LimitReader(opened, 1024*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode backup manifest: %w", err)
	}
	if manifest.Files == nil {
		return Manifest{}, errors.New("backup manifest has no files")
	}
	return manifest, nil
}

func extractFile(source *zip.File, destination string) (resultErr error) {
	opened, err := source.Open()
	if err != nil {
		return err
	}
	defer opened.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := output.Close(); resultErr == nil {
			resultErr = closeErr
		}
	}()
	_, err = io.Copy(output, opened)
	return err
}

func validateDatabase(path, requiredTable string) error {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()
	var integrity string
	if err := db.QueryRow(`PRAGMA integrity_check`).Scan(&integrity); err != nil {
		return err
	}
	if integrity != "ok" {
		return fmt.Errorf("integrity check: %s", integrity)
	}
	var table string
	if err := db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, requiredTable,
	).Scan(&table); err != nil {
		return fmt.Errorf("required table %s: %w", requiredTable, err)
	}
	return nil
}

func (s *Service) replaceDatabases(extracted string, manifest Manifest) error {
	preRestore, err := s.preRestoreDirectory()
	if err != nil {
		return err
	}
	livePaths := map[string]string{
		"user.sqlite":    s.paths.UserDB,
		"library.sqlite": s.paths.LibraryDB,
	}
	type movedPath struct {
		backup string
		live   string
	}
	moved := []movedPath{}
	installed := []string{}
	rollback := func(cause error) error {
		var rollbackErrors []error
		for _, name := range installed {
			if err := os.Remove(livePaths[name]); err != nil {
				rollbackErrors = append(rollbackErrors, err)
			}
		}
		for index := len(moved) - 1; index >= 0; index-- {
			value := moved[index]
			if err := os.Rename(value.backup, value.live); err != nil {
				rollbackErrors = append(rollbackErrors, err)
			}
		}
		if len(rollbackErrors) == 0 {
			if err := os.RemoveAll(preRestore); err != nil {
				rollbackErrors = append(rollbackErrors, err)
			}
		}
		if len(rollbackErrors) > 0 {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("pre-restore data retained at %s", preRestore))
		}
		return errors.Join(append([]error{cause}, rollbackErrors...)...)
	}
	for _, name := range sortedKeys(manifest.Files) {
		live := livePaths[name]
		if _, err := os.Stat(live); err == nil {
			backupPath := filepath.Join(preRestore, name)
			if err := os.Rename(live, backupPath); err != nil {
				return rollback(err)
			}
			moved = append(moved, movedPath{backup: backupPath, live: live})
		} else if !errors.Is(err, os.ErrNotExist) {
			return rollback(err)
		}
		for _, suffix := range []string{"-wal", "-shm"} {
			sidecar := live + suffix
			if _, err := os.Stat(sidecar); err == nil {
				backupName := name + suffix
				backupPath := filepath.Join(preRestore, backupName)
				if err := os.Rename(sidecar, backupPath); err != nil {
					return rollback(err)
				}
				moved = append(moved, movedPath{backup: backupPath, live: sidecar})
			}
		}
	}
	for _, name := range sortedKeys(manifest.Files) {
		live := livePaths[name]
		if err := os.Rename(filepath.Join(extracted, name), live); err != nil {
			return rollback(err)
		}
		if err := os.Chmod(live, 0o600); err != nil {
			return rollback(err)
		}
		installed = append(installed, name)
	}
	return nil
}

func (s *Service) preRestoreDirectory() (string, error) {
	base := "pre-restore-" + s.now().Format("20060102-150405")
	for suffix := 0; ; suffix++ {
		name := base
		if suffix > 0 {
			name = fmt.Sprintf("%s-%d", base, suffix)
		}
		path := filepath.Join(s.paths.BackupsDir, name)
		if err := os.Mkdir(path, 0o700); err == nil {
			return path, nil
		} else if !errors.Is(err, os.ErrExist) {
			return "", err
		}
	}
}

func fileDigest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
