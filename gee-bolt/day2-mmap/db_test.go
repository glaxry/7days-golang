package geebolt

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenSyncAndClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	want := []byte("geebolt")

	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	copy(db.data, want)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got[:len(want)], want) {
		t.Fatalf("mapped data = %q, want %q", got[:len(want)], want)
	}

	db, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if !bytes.Equal(db.data[:len(want)], want) {
		t.Fatalf("reopened data = %q, want %q", db.data[:len(want)], want)
	}
}
