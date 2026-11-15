// Command linkctl is the Linkforge operator CLI.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/avekrivoy/linkforge/internal/config"
	"github.com/avekrivoy/linkforge/internal/link"
	"github.com/avekrivoy/linkforge/internal/linkio"
	"github.com/avekrivoy/linkforge/internal/store"
)

// version is overwritten at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	code := run(ctx, os.Args[1:], os.LookupEnv, os.Stdout, os.Stderr)

	stop()
	os.Exit(code)
}

func run(
	ctx context.Context,
	args []string,
	env func(string) (string, bool),
	stdout, stderr io.Writer,
) int {
	logLevel := "info"

	if value, ok := env("LF_LOG_LEVEL"); ok && strings.TrimSpace(value) != "" {
		logLevel = value
	}

	level, err := config.ParseLevel(logLevel)

	if err != nil {
		fmt.Fprintln(stderr, "invalid LF_LOG_LEVEL")
		return 2
	}

	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{
		Level: level,
	}))

	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}

	switch args[0] {
	case "import":
		return runImport(ctx, args[1:], stdout, stderr, logger)

	case "export":
		return runExport(ctx, args[1:], stdout, stderr, logger)

	case "config":
		return runConfig(ctx, args[1:], env, stdout, stderr, logger)

	case "version":
		return runVersion(args[1:], stdout, stderr)

	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: linkctl <import|export|config|version>")
}

func runImport(
	ctx context.Context,
	args []string,
	stdout, stderr io.Writer,
	logger *slog.Logger,
) int {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(stderr)

	input := fs.String("in", "", "input targets file")
	db := fs.String("db", "", "database file")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if fs.NArg() != 0 || *input == "" || *db == "" {
		fmt.Fprintln(stderr, "usage: linkctl import -in FILE -db FILE")
		return 2
	}

	return importLinks(ctx, *input, *db, stdout, logger)
}

func runExport(
	ctx context.Context,
	args []string,
	stdout, stderr io.Writer,
	logger *slog.Logger,
) int {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(stderr)

	db := fs.String("db", "", "database file")
	out := fs.String("out", "", "output CSV file")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if fs.NArg() != 0 || *db == "" {
		fmt.Fprintln(stderr, "usage: linkctl export -db FILE [-out FILE]")
		return 2
	}

	if err := ctx.Err(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	links, err := linkio.LoadFile(*db)
	if err != nil {
		fmt.Fprintf(stderr, "cannot load database: %v\n", err)
		return 1
	}

	if *out == "" {
		if err := linkio.WriteCSV(stdout, links); err != nil {
			fmt.Fprintf(stderr, "cannot write CSV: %v\n", err)
			return 1
		}
		return 0
	}

	file, err := os.Create(*out)
	if err != nil {
		fmt.Fprintf(stderr, "cannot create output file: %v\n", err)
		return 1
	}

	writeErr := linkio.WriteCSV(file, links)
	closeErr := file.Close()

	if writeErr != nil {
		fmt.Fprintf(stderr, "cannot write CSV: %v\n", writeErr)
	}
	if closeErr != nil {
		fmt.Fprintf(stderr, "cannot close output file: %v\n", closeErr)
	}

	if writeErr != nil || closeErr != nil {
		return 1
	}

	logger.Info("export finished", "count", len(links))
	return 0
}

func runConfig(
	ctx context.Context,
	args []string,
	env func(string) (string, bool),
	stdout, stderr io.Writer,
	logger *slog.Logger,
) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "usage: linkctl config")
		return 2
	}

	loader := config.NewEnvLoader(env)
	cfg, err := loader.Load(ctx)

	if err != nil {
		fmt.Fprintf(stderr, "invalid configuration:\n%v\n", err)
		return 1
	}

	logger.Info("configuration loaded", "config", cfg)

	fmt.Fprintln(stdout, "config ok")
	return 0
}

func runVersion(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "usage: linkctl version")
		return 2
	}

	fmt.Fprintf(stdout, "linkctl %s\n", version)
	return 0
}

func importLinks(
	ctx context.Context,
	inputPath, dbPath string,
	stdout io.Writer,
	logger *slog.Logger,
) int {
	if err := ctx.Err(); err != nil {
		logger.Error("import cancelled", "error", err)
		return 1
	}

	logger.Info("import started")

	inputFile, err := os.Open(inputPath)

	if err != nil {
		logger.Error("cannot open input file", "error", err)
		return 1
	}

	targets, readErr := linkio.ReadTargets(inputFile)
	closeErr := inputFile.Close()

	if readErr != nil {
		logger.Error("cannot read targets", "error", readErr)
		return 1
	}

	if closeErr != nil {
		logger.Error("cannot close input file", "error", closeErr)
		return 1
	}

	links, err := linkio.LoadFile(dbPath)

	if err != nil && !errors.Is(err, os.ErrNotExist) {
		logger.Error("cannot load database", "error", err)
		return 1
	}
	if errors.Is(err, os.ErrNotExist) {
		links = nil
	}

	var maxID uint64

	for _, l := range links {
		if l.ID > maxID {
			maxID = l.ID
		}
	}

	if maxID == ^uint64(0) {
		logger.Error("cannot allocate next link ID")
		return 1
	}

	st := store.New()

	for _, l := range links {
		if _, err := st.Create(ctx, l); err != nil {
			logger.Error("cannot restore link", "id", l.ID, "error", err)
			return 1
		}
	}

	service := link.NewService(st, link.WithFirstID(maxID+1))

	imported := 0
	skipped := 0

	for _, target := range targets {
		if err := ctx.Err(); err != nil {
			logger.Error("import cancelled", "error", err)
			return 1
		}

		created, err := service.Shorten(ctx, link.ShortenRequest{
			Target:  target.URL,
			OwnerID: "cli",
		})

		if err != nil {
			skipped++
			logger.Warn(
				"target skipped",
				"line", target.Line,
				"error", err,
			)
			continue
		}

		imported++

		logger.Debug(
			"target imported",
			"line", target.Line,
			"id", created.ID,
		)
	}

	if err := ctx.Err(); err != nil {
		logger.Error("import cancelled", "error", err)
		return 1
	}

	allLinks := st.All()

	if err := linkio.SaveFile(dbPath, allLinks); err != nil {
		logger.Error("cannot save database", "error", err)
		return 1
	}

	if _, err := fmt.Fprintf(stdout, "imported: %d\n", imported); err != nil {
		logger.Error("cannot write import result", "error", err)
		return 1
	}

	if _, err := fmt.Fprintf(stdout, "skipped: %d\n", skipped); err != nil {
		logger.Error("cannot write import result", "error", err)
		return 1
	}

	logger.Info(
		"import finished",
		"imported", imported,
		"skipped", skipped,
	)

	if skipped > 0 {
		return 1
	}

	return 0
}
