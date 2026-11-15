package linkio

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/avekrivoy/linkforge/internal/link"
)

func assertLinksEqual(t *testing.T, got, want []link.Link) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("links = %#v, want %#v", got, want)
	}
}

func TestReadTargets(t *testing.T) {
	input := `
# comment
https://example.com

  https://golang.org
   # another comment
https://github.com
`

	got, err := ReadTargets(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ReadTargets() error = %v", err)
	}

	want := []Target{
		{Line: 3, URL: "https://example.com"},
		{Line: 5, URL: "https://golang.org"},
		{Line: 7, URL: "https://github.com"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("ReadTargets() = %#v, want %#v", got, want)
	}
}

func TestReadTargetsEmpty(t *testing.T) {
	input := " \n# comment\n\n"

	got, err := ReadTargets(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ReadTargets() error = %v", err)
	}

	if len(got) != 0 {
		t.Errorf("ReadTargets() returned %d targets, want 0", len(got))
	}
}

func TestReadTargetsError(t *testing.T) {
	_, err := ReadTargets(errorReader{})
	if err == nil {
		t.Fatal("ReadTargets() expected an error, got nil")
	}

	if !strings.Contains(err.Error(), "read targets") {
		t.Errorf("error = %q, want read targets context", err)
	}
}

func TestReadLinksWriteLinksRoundTrip(t *testing.T) {
	want := []link.Link{
		{
			ID:      1,
			OwnerID: "user-1",
			Code:    "abc",
			Target:  "https://example.com",
		},
		{
			ID:      2,
			OwnerID: "user-2",
			Code:    "def",
			Target:  "https://golang.org",
		},
	}

	var buf bytes.Buffer

	if err := WriteLinks(&buf, want); err != nil {
		t.Fatalf("WriteLinks() error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != len(want) {
		t.Fatalf("got %d JSON lines, want %d", len(lines), len(want))
	}

	for i, line := range lines {
		var l link.Link

		if err := json.Unmarshal([]byte(line), &l); err != nil {
			t.Fatalf("line %d is not valid JSON: %v", i+1, err)
		}
	}

	got, err := ReadLinks(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatalf("ReadLinks() error = %v", err)
	}

	assertLinksEqual(t, got, want)
}

func TestReadLinksBadRecord(t *testing.T) {
	input := `{"ID":1,"Code":"abc","Target":"https://example.com"}
not-json
{"ID":3,"Code":"ghi","Target":"https://golang.org"}`

	_, err := ReadLinks(strings.NewReader(input))
	if err == nil {
		t.Fatal("ReadLinks() expected an error, got nil")
	}

	if !strings.Contains(err.Error(), "record 2") {
		t.Errorf("error = %q, want record 2", err)
	}
}

func TestReadLinksError(t *testing.T) {
	_, err := ReadLinks(errorReader{})
	if err == nil {
		t.Fatal("ReadLinks() expected an error, got nil")
	}

	if !strings.Contains(err.Error(), "read links") {
		t.Errorf("error = %q, want read links context", err)
	}
}

func TestWriteLinksError(t *testing.T) {
	err := WriteLinks(errorWriter{}, []link.Link{
		{ID: 1, Code: "abc"},
	})

	if err == nil {
		t.Fatal("WriteLinks() expected an error, got nil")
	}

	if !strings.Contains(err.Error(), "write link") {
		t.Errorf("error = %q, want write link context", err)
	}
}

func TestWriteCSVExactText(t *testing.T) {
	createdAt := time.Date(
		2026, 11, 15, 12, 30, 0, 0,
		time.FixedZone("UTC+2", 2*60*60),
	)

	expiresAt := time.Date(
		2026, 12, 15, 10, 0, 0, 0,
		time.UTC,
	)

	links := []link.Link{
		{
			ID:        1,
			Code:      "abc",
			Target:    "https://example.com",
			OwnerID:   "user-1",
			CreatedAt: createdAt,
			ExpiresAt: expiresAt,
		},
		{
			ID:        2,
			Code:      "def",
			Target:    "https://golang.org",
			OwnerID:   "user-2",
			CreatedAt: createdAt,
			ExpiresAt: time.Time{},
		},
	}

	var buf bytes.Buffer

	if err := WriteCSV(&buf, links); err != nil {
		t.Fatalf("WriteCSV() error = %v", err)
	}

	want := "" +
		"id,code,target,owner_id,created_at,expires_at\n" +
		"1,abc,https://example.com,user-1,2026-11-15T10:30:00Z,2026-12-15T10:00:00Z\n" +
		"2,def,https://golang.org,user-2,2026-11-15T10:30:00Z,\n"

	if got := buf.String(); got != want {
		t.Errorf("WriteCSV() output:\n%q\nwant:\n%q", got, want)
	}
}

func TestWriteCSVError(t *testing.T) {
	err := WriteCSV(errorWriter{}, nil)

	if err == nil {
		t.Fatal("WriteCSV() expected an error, got nil")
	}
}

func TestSaveFileAndLoadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "links.jsonl")

	want := []link.Link{
		{
			ID:      1,
			Code:    "abc",
			Target:  "https://example.com",
			OwnerID: "user-1",
		},
		{
			ID:      2,
			Code:    "def",
			Target:  "https://golang.org",
			OwnerID: "user-2",
		},
	}

	if err := SaveFile(path, want); err != nil {
		t.Fatalf("SaveFile() error = %v", err)
	}

	got, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	assertLinksEqual(t, got, want)
}

func TestLoadFileMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.jsonl")

	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("LoadFile() expected an error, got nil")
	}

	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error = %v, want fs.ErrNotExist", err)
	}

	if !strings.Contains(err.Error(), path) {
		t.Errorf("error %q does not contain path %q", err, path)
	}
}

func TestLoadFileInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.jsonl")

	if err := os.WriteFile(path, []byte("not-json\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("LoadFile() expected an error, got nil")
	}

	if !strings.Contains(err.Error(), path) {
		t.Errorf("error %q does not contain path %q", err, path)
	}

	if !strings.Contains(err.Error(), "record 1") {
		t.Errorf("error = %q, want record 1", err)
	}
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}
