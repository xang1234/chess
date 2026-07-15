package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
)

type IntegrityError struct {
	Path   string `json:"path"`
	Detail string `json:"detail"`
}

func (e *IntegrityError) Error() string {
	return fmt.Sprintf("database integrity check failed for %s: %s", e.Path, e.Detail)
}

func CheckDurableIntegrity(paths Paths) error {
	for _, path := range []string{paths.UserDB, paths.LibraryDB} {
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return err
		}
		db, err := sql.Open("sqlite", path)
		if err != nil {
			return &IntegrityError{Path: path, Detail: err.Error()}
		}
		var detail string
		checkErr := db.QueryRow(`PRAGMA quick_check`).Scan(&detail)
		closeErr := db.Close()
		if checkErr != nil {
			return &IntegrityError{Path: path, Detail: checkErr.Error()}
		}
		if closeErr != nil {
			return closeErr
		}
		if detail != "ok" {
			return &IntegrityError{Path: path, Detail: detail}
		}
	}
	return nil
}
