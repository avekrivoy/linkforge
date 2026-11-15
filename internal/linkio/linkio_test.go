package linkio

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/avekrivoy/linkforge/internal/link"
)

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
	got, err := ReadTargets(strings.NewReader(" \n# comment\n\n"))
	if err != nil {
		t.Fatalf("ReadTargets() error = %v", err)
	}

	if len(got) != 0 {
		t.Errorf("ReadTargets() returned %d targets, want 0", len(got))
	}
}

func TestReadLinks(t *testing.T) {
	input := strings.Join([]string{
		`{"ID":1,"OwnerID":"user-1","Code":"abc","Target":"https://example.com"}`,
		`{"ID":2,"OwnerID":"user-2","Code":"def","Target":"https://golang.org"}`,
	}, "\n")

	got, err := ReadLinks(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ReadLinks() error = %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("ReadLinks() returned %d links, want 2", len(got))
	}

	if got[0].ID != 1 || got[0].Code != "abc" {
		t.Errorf("first link = %+v", got[0])
	}

	if got[1].ID != 2 || got[1].Code != "def" {
		t.Errorf("second link = %+v", got[1])
	}
}

func TestReadLinksInvalidJSON(t *testing.T) {
	input := `{"ID":1,"Code":"abc"}
not-json`

	_, err := ReadLinks(strings.NewReader(input))
	if err == nil {
		t.Fatal("ReadLinks() expected an error, got nil")
	}

	if !strings.Contains(err.Error(), "record 2") {
		t.Errorf("error = %q, want record number", err)
	}
}

func TestWriteLinks(t *testing.T) {
	want := []link.Link{
		{ID: 1, Code: "abc", Target: "https://example.com"},
		{ID: 2, Code: "def", Target: "https://golang.org"},
	}

	var buf bytes.Buffer

	if err := WriteLinks(&buf, want); err != nil {
		t.Fatalf("WriteLinks() error = %v", err)
	}

	got, err := ReadLinks(&buf)
	if err != nil {
		t.Fatalf("ReadLinks() error = %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("round trip = %#v, want %#v", got, want)
	}
}

func TestWriteCSV(t *testing.T) {
	createdAt := time.Date(2026, 11, 15, 12, 30, 0, 0, time.FixedZone("UTC+2", 2*60*60))
	expiresAt := time.Date(2026, 12, 15, 10, 0, 0, 0, time.UTC)

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

	records, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatalf("reading CSV: %v", err)
	}

	if len(records) != 3 {
		t.Fatalf("got %d CSV records, want 3", len(records))
	}

	wantHeader := []string{
		"id", "code", "target", "owner_id", "created_at", "expires_at",
	}
	if !reflect.DeepEqual(records[0], wantHeader) {
		t.Errorf("header = %#v, want %#v", records[0], wantHeader)
	}

	if records[1][0] != "1" ||
		records[1][1] != "abc" ||
		records[1][5] != expiresAt.Format(time.RFC3339) {
		t.Errorf("first CSV record = %#v", records[1])
	}

	if records[2][0] != "2" || records[2][5] != "" {
		t.Errorf("second CSV record = %#v, want empty expires_at", records[2])
	}

	wantCreatedAt := createdAt.UTC().Format(time.RFC3339)
	if records[1][4] != wantCreatedAt {
		t.Errorf("created_at = %q, want %q", records[1][4], wantCreatedAt)
	}
}

func TestLoadFileNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.jsonl")

	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("LoadFile() expected an error, got nil")
	}

	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error = %v, want fs.ErrNotExist", err)
	}
}

func TestSaveFileAndLoadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "links.jsonl")

	want := []link.Link{
		{ID: 1, Code: "abc", Target: "https://example.com"},
		{ID: 2, Code: "def", Target: "https://golang.org"},
	}

	if err := SaveFile(path, want); err != nil {
		t.Fatalf("SaveFile() error = %v", err)
	}

	got, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile() error = %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("LoadFile() = %#v, want %#v", got, want)
	}
}

func TestLoadFileInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.jsonl")

	if err := os.WriteFile(path, []byte("not-json\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("LoadFile() expected an error, got nil")
	}

	if !strings.Contains(err.Error(), path) {
		t.Errorf("error %q does not contain path %q", err, path)
	}
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
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

func TestWriteCSVError(t *testing.T) {
	err := WriteCSV(errorWriter{}, nil)

	if err == nil {
		t.Fatal("WriteCSV() expected an error, got nil")
	}
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
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

func TestReadLinksError(t *testing.T) {
	_, err := ReadLinks(errorReader{})
	if err == nil {
		t.Fatal("ReadLinks() expected an error, got nil")
	}

	if !strings.Contains(err.Error(), "read links") {
		t.Errorf("error = %q, want read links context", err)
	}
}

func TestWriteLinksJSONLines(t *testing.T) {
	links := []link.Link{
		{ID: 1, Code: "abc"},
		{ID: 2, Code: "def"},
	}

	var buf bytes.Buffer
	if err := WriteLinks(&buf, links); err != nil {
		t.Fatal(err)
	}

	scanner := bufio.NewScanner(&buf)
	count := 0

	for scanner.Scan() {
		var l link.Link
		if err := json.Unmarshal(scanner.Bytes(), &l); err != nil {
			t.Fatalf("line is not valid JSON: %v", err)
		}
		count++
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		t.Fatal(err)
	}

	if count != len(links) {
		t.Errorf("got %d JSON lines, want %d", count, len(links))
	}
}
