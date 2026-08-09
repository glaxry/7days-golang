package geebolt

import (
	"errors"
	"fmt"
	"os"
)

type DB struct {
	data    []byte
	file    *os.File
	mapping mmapState
}

const maxMapSize int64 = 1 << 31

func (db *DB) mmap(sz int) error {
	if sz <= 0 || int64(sz) > maxMapSize {
		return fmt.Errorf("geebolt: invalid mmap size %d", sz)
	}
	if len(db.data) > 0 {
		if err := unmapFile(db.data, db.mapping); err != nil {
			return fmt.Errorf("geebolt: unmap existing region: %w", err)
		}
		db.data = nil
		db.mapping = mmapState{}
	}
	if err := db.file.Truncate(int64(sz)); err != nil {
		return fmt.Errorf("geebolt: resize database: %w", err)
	}

	data, mapping, err := mapFile(db.file, sz)
	if err != nil {
		return fmt.Errorf("geebolt: mmap database: %w", err)
	}
	db.data = data
	db.mapping = mapping
	return nil
}

// Open opens path and maps its contents into memory. A new database starts
// with one operating-system page so that it can be mapped on every platform.
func Open(path string) (*DB, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("geebolt: open database: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("geebolt: stat database: %w", err)
	}
	size := info.Size()
	if size == 0 {
		size = int64(os.Getpagesize())
	}
	if size > maxMapSize {
		_ = file.Close()
		return nil, fmt.Errorf("geebolt: database is too large: %d bytes", size)
	}

	db := &DB{file: file}
	if err := db.mmap(int(size)); err != nil {
		_ = file.Close()
		return nil, err
	}
	return db, nil
}

// Sync flushes changes in the mapped region to disk.
func (db *DB) Sync() error {
	if db == nil || len(db.data) == 0 {
		return nil
	}
	if err := syncMappedFile(db.data, db.mapping); err != nil {
		return fmt.Errorf("geebolt: sync database: %w", err)
	}
	return nil
}

// Close flushes and releases the mapping before closing the file.
func (db *DB) Close() error {
	if db == nil {
		return nil
	}
	var errs []error
	if len(db.data) > 0 {
		errs = append(errs, db.Sync(), unmapFile(db.data, db.mapping))
		db.data = nil
		db.mapping = mmapState{}
	}
	if db.file != nil {
		errs = append(errs, db.file.Close())
		db.file = nil
	}
	return errors.Join(errs...)
}
