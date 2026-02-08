//go:build !integration

package beater

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadTokenFile(t *testing.T) {
	dir, err := os.MkdirTemp("", "tokenstore-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "tokens.json")

	original := StoredToken{
		AccessToken:  "access-abc",
		RefreshToken: "refresh-xyz",
		ExpiresIn:    10800,
		ObtainedAt:   1700000000,
		Scope:        []string{"read_station"},
	}

	if err := SaveTokenFile(path, original); err != nil {
		t.Fatalf("SaveTokenFile failed: %v", err)
	}

	loaded, err := LoadTokenFile(path)
	if err != nil {
		t.Fatalf("LoadTokenFile failed: %v", err)
	}

	if loaded.AccessToken != original.AccessToken {
		t.Errorf("AccessToken: got %q, want %q", loaded.AccessToken, original.AccessToken)
	}
	if loaded.RefreshToken != original.RefreshToken {
		t.Errorf("RefreshToken: got %q, want %q", loaded.RefreshToken, original.RefreshToken)
	}
	if loaded.ExpiresIn != original.ExpiresIn {
		t.Errorf("ExpiresIn: got %d, want %d", loaded.ExpiresIn, original.ExpiresIn)
	}
	if loaded.ObtainedAt != original.ObtainedAt {
		t.Errorf("ObtainedAt: got %d, want %d", loaded.ObtainedAt, original.ObtainedAt)
	}
	if len(loaded.Scope) != 1 || loaded.Scope[0] != "read_station" {
		t.Errorf("Scope: got %v, want [read_station]", loaded.Scope)
	}
}

func TestSaveTokenFilePermissions(t *testing.T) {
	dir, err := os.MkdirTemp("", "tokenstore-perm-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "tokens.json")

	token := StoredToken{
		AccessToken:  "a",
		RefreshToken: "r",
	}

	if err := SaveTokenFile(path, token); err != nil {
		t.Fatalf("SaveTokenFile failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("file permissions: got %04o, want 0600", perm)
	}
}

func TestLoadTokenFileMissing(t *testing.T) {
	_, err := LoadTokenFile("/nonexistent/path/tokens.json")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestLoadTokenFileInvalidJSON(t *testing.T) {
	dir, err := os.MkdirTemp("", "tokenstore-invalid-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "tokens.json")
	if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err = LoadTokenFile(path)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestSaveTokenFileOverwrite(t *testing.T) {
	dir, err := os.MkdirTemp("", "tokenstore-overwrite-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "tokens.json")

	first := StoredToken{RefreshToken: "first"}
	second := StoredToken{RefreshToken: "second"}

	if err := SaveTokenFile(path, first); err != nil {
		t.Fatalf("first save failed: %v", err)
	}
	if err := SaveTokenFile(path, second); err != nil {
		t.Fatalf("second save failed: %v", err)
	}

	loaded, err := LoadTokenFile(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.RefreshToken != "second" {
		t.Errorf("RefreshToken: got %q, want %q", loaded.RefreshToken, "second")
	}
}
