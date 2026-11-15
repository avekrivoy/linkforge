package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/avekrivoy/linkforge/internal/config"
	"github.com/avekrivoy/linkforge/internal/link"
	"github.com/avekrivoy/linkforge/internal/linkio"
)

func testEnv(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func validTestEnv() map[string]string {
	return map[string]string{
		"LF_PG_DSN":     "postgres://localhost/test",
		"LF_MONGO_URI":  "mongodb://localhost/test",
		"LF_JWT_SECRET": "this-is-a-test-secret-with-at-least-32-bytes",
	}
}

func runCommand(
	t *testing.T,
	ctx context.Context,
	args []string,
	env map[string]string,
) (int, string, string) {
	t.Helper()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run(ctx, args, testEnv(env), &stdout, &stderr)

	return code, stdout.String(), stderr.String()
}

func writeTargetsFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}

	return path
}

func readLinks(t *testing.T, path string) []link.Link {
	t.Helper()

	links, err := linkio.LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile(%q) error = %v", path, err)
	}

	return links
}

func TestImportThenExport(t *testing.T) {
	dir := t.TempDir()

	inputPath := writeTargetsFile(
		t,
		dir,
		"targets.txt",
		"https://example.com\nhttps://golang.org\n",
	)

	dbPath := filepath.Join(dir, "links.jsonl")

	env := validTestEnv()

	code, stdout, stderr := runCommand(
		t,
		context.Background(),
		[]string{"import", "-in", inputPath, "-db", dbPath},
		env,
	)

	if code != 0 {
		t.Fatalf("import exit code = %d, want 0; stderr: %s", code, stderr)
	}

	if !strings.Contains(stdout, "imported: 2") {
		t.Errorf("import stdout = %q, want imported: 2", stdout)
	}

	if !strings.Contains(stdout, "skipped: 0") {
		t.Errorf("import stdout = %q, want skipped: 0", stdout)
	}

	links := readLinks(t, dbPath)

	if len(links) != 2 {
		t.Fatalf("database contains %d links, want 2", len(links))
	}

	// Експортуємо базу в CSV через stdout.
	code, stdout, stderr = runCommand(
		t,
		context.Background(),
		[]string{"export", "-db", dbPath},
		env,
	)

	if code != 0 {
		t.Fatalf("export exit code = %d, want 0; stderr: %s", code, stderr)
	}

	for _, expected := range []string{
		"id,code,target,owner_id,created_at,expires_at",
		"https://example.com",
		"https://golang.org",
	} {
		if !strings.Contains(stdout, expected) {
			t.Errorf("export output does not contain %q:\n%s", expected, stdout)
		}
	}
}

func TestSecondImportContinuesIDNumbering(t *testing.T) {
	dir := t.TempDir()

	firstInput := writeTargetsFile(
		t,
		dir,
		"first.txt",
		"https://example.com\nhttps://golang.org\n",
	)

	secondInput := writeTargetsFile(
		t,
		dir,
		"second.txt",
		"https://github.com\nhttps://go.dev\n",
	)

	dbPath := filepath.Join(dir, "links.jsonl")
	env := validTestEnv()

	code, _, stderr := runCommand(
		t,
		context.Background(),
		[]string{"import", "-in", firstInput, "-db", dbPath},
		env,
	)

	if code != 0 {
		t.Fatalf("first import exit code = %d; stderr: %s", code, stderr)
	}

	firstLinks := readLinks(t, dbPath)
	if len(firstLinks) != 2 {
		t.Fatalf("first import saved %d links, want 2", len(firstLinks))
	}

	firstMaxID := firstLinks[len(firstLinks)-1].ID

	code, _, stderr = runCommand(
		t,
		context.Background(),
		[]string{"import", "-in", secondInput, "-db", dbPath},
		env,
	)

	if code != 0 {
		t.Fatalf("second import exit code = %d; stderr: %s", code, stderr)
	}

	allLinks := readLinks(t, dbPath)

	if len(allLinks) != 4 {
		t.Fatalf("database contains %d links, want 4", len(allLinks))
	}

	if allLinks[2].ID != firstMaxID+1 {
		t.Errorf("third link ID = %d, want %d", allLinks[2].ID, firstMaxID+1)
	}

	if allLinks[3].ID != firstMaxID+2 {
		t.Errorf("fourth link ID = %d, want %d", allLinks[3].ID, firstMaxID+2)
	}
}

func TestImportSkipsBadTargetAndSavesGoodOnes(t *testing.T) {
	dir := t.TempDir()

	inputPath := writeTargetsFile(
		t,
		dir,
		"targets.txt",
		"https://example.com\nnot-a-valid-url\nhttps://golang.org\n",
	)

	dbPath := filepath.Join(dir, "links.jsonl")
	env := validTestEnv()

	code, stdout, stderr := runCommand(
		t,
		context.Background(),
		[]string{"import", "-in", inputPath, "-db", dbPath},
		env,
	)

	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr: %s", code, stderr)
	}

	if !strings.Contains(stdout, "imported: 2") {
		t.Errorf("stdout = %q, want imported: 2", stdout)
	}

	if !strings.Contains(stdout, "skipped: 1") {
		t.Errorf("stdout = %q, want skipped: 1", stdout)
	}

	if !strings.Contains(stderr, "target skipped") {
		t.Errorf("stderr = %q, want target skipped log", stderr)
	}

	if !strings.Contains(stderr, "line=2") {
		t.Errorf("stderr = %q, want line=2 attribute", stderr)
	}

	links := readLinks(t, dbPath)

	if len(links) != 2 {
		t.Fatalf("database contains %d links, want 2", len(links))
	}

	if links[0].Target != "https://example.com" {
		t.Errorf("first target = %q", links[0].Target)
	}

	if links[1].Target != "https://golang.org" {
		t.Errorf("second target = %q", links[1].Target)
	}
}

func TestImportCanceledContextLeavesDatabaseUntouched(t *testing.T) {
	dir := t.TempDir()

	inputPath := writeTargetsFile(
		t,
		dir,
		"targets.txt",
		"https://example.com\n",
	)

	dbPath := filepath.Join(dir, "links.jsonl")

	initialLinks := []link.Link{
		{
			ID:      10,
			Code:    "existing",
			Target:  "https://existing.example.com",
			OwnerID: "test",
		},
	}

	if err := linkio.SaveFile(dbPath, initialLinks); err != nil {
		t.Fatalf("SaveFile() error = %v", err)
	}

	before, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	code, _, _ := runCommand(
		t,
		ctx,
		[]string{"import", "-in", inputPath, "-db", dbPath},
		validTestEnv(),
	)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}

	after, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("ReadFile() after import error = %v", err)
	}

	if !bytes.Equal(after, before) {
		t.Errorf("database changed after canceled import")
	}
}

func TestRunUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "no command",
			args: nil,
		},
		{
			name: "unknown command",
			args: []string{"unknown"},
		},
		{
			name: "import missing input",
			args: []string{"import", "-db", "db.jsonl"},
		},
		{
			name: "import missing database",
			args: []string{"import", "-in", "targets.txt"},
		},
		{
			name: "import missing all flags",
			args: []string{"import"},
		},
		{
			name: "import unknown flag",
			args: []string{"import", "-unknown", "value"},
		},
		{
			name: "import positional argument",
			args: []string{"import", "-in", "in.txt", "-db", "db.jsonl", "extra"},
		},
		{
			name: "export missing database",
			args: []string{"export"},
		},
		{
			name: "export unknown flag",
			args: []string{"export", "-unknown", "value"},
		},
		{
			name: "export positional argument",
			args: []string{"export", "-db", "db.jsonl", "extra"},
		},
		{
			name: "config extra argument",
			args: []string{"config", "extra"},
		},
		{
			name: "version extra argument",
			args: []string{"version", "extra"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, _ := runCommand(
				t,
				context.Background(),
				tt.args,
				validTestEnv(),
			)

			if code != 2 {
				t.Errorf("exit code = %d, want 2", code)
			}
		})
	}
}

func TestRunInvalidLogLevel(t *testing.T) {
	env := validTestEnv()
	env["LF_LOG_LEVEL"] = "trace"

	code, stdout, stderr := runCommand(
		t,
		context.Background(),
		[]string{"version"},
		env,
	)

	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}

	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}

	if !strings.Contains(stderr, "invalid LF_LOG_LEVEL") {
		t.Errorf("stderr = %q, want invalid LF_LOG_LEVEL", stderr)
	}
}

func TestRunConfigGoodEnvironment(t *testing.T) {
	env := validTestEnv()

	code, stdout, stderr := runCommand(
		t,
		context.Background(),
		[]string{"config"},
		env,
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr)
	}

	if !strings.Contains(stdout, "config ok") {
		t.Errorf("stdout = %q, want config ok", stdout)
	}

	if !strings.Contains(stderr, "configuration loaded") {
		t.Errorf("stderr = %q, want configuration loaded log", stderr)
	}

	for _, secret := range []string{
		env["LF_PG_DSN"],
		env["LF_MONGO_URI"],
		env["LF_JWT_SECRET"],
	} {
		if strings.Contains(stderr, secret) {
			t.Errorf("configuration log exposes secret %q", secret)
		}
	}
}

func TestRunConfigBadEnvironment(t *testing.T) {
	env := validTestEnv()
	env["LF_JWT_SECRET"] = "short"

	code, stdout, stderr := runCommand(
		t,
		context.Background(),
		[]string{"config"},
		env,
	)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}

	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}

	if !strings.Contains(stderr, "invalid configuration") {
		t.Errorf("stderr = %q, want invalid configuration", stderr)
	}

	if strings.Contains(stderr, env["LF_PG_DSN"]) {
		t.Errorf("error output exposes PostgreSQL DSN")
	}
}

func TestRunConfigCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	code, stdout, stderr := runCommand(
		t,
		ctx,
		[]string{"config"},
		validTestEnv(),
	)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}

	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}

	if !strings.Contains(stderr, context.Canceled.Error()) {
		t.Errorf("stderr = %q, want canceled context error", stderr)
	}
}

func TestRunVersion(t *testing.T) {
	code, stdout, stderr := runCommand(
		t,
		context.Background(),
		[]string{"version"},
		validTestEnv(),
	)

	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr: %s", code, stderr)
	}

	if stdout != "linkctl "+version+"\n" {
		t.Errorf("stdout = %q, want %q", stdout, "linkctl "+version+"\n")
	}

	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

func TestTestEnvDoesNotReadRealEnvironment(t *testing.T) {
	env := testEnv(map[string]string{
		"LF_LOG_LEVEL": "debug",
	})

	value, ok := env("LF_LOG_LEVEL")
	if !ok || value != "debug" {
		t.Fatalf("env lookup = (%q, %v), want (debug, true)", value, ok)
	}

	if _, ok := env("VARIABLE_THAT_DOES_NOT_EXIST"); ok {
		t.Fatal("unexpected environment variable found")
	}
}

func TestConfigParseLevelErrorIsInvalid(t *testing.T) {
	_, err := config.ParseLevel("not-a-level")
	if err == nil {
		t.Fatal("ParseLevel() expected an error")
	}

	if !errors.Is(err, config.ErrInvalid) {
		t.Errorf("error = %v, want config.ErrInvalid", err)
	}
}
