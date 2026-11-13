package linkio

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/avekrivoy/linkforge/internal/link"
)

type Target struct {
	Line int
	URL  string
}

func ReadTargets(r io.Reader) ([]Target, error) {
	scanner := bufio.NewScanner(r)

	var targets []Target
	line := 0

	for scanner.Scan() {
		line++

		value := strings.TrimSpace(scanner.Text())

		if value == "" || strings.HasPrefix(value, "#") {
			continue
		}

		targets = append(targets, Target{
			Line: line,
			URL:  value,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read targets: %w", err)
	}

	return targets, nil
}

func ReadLinks(r io.Reader) ([]link.Link, error) {
	scanner := bufio.NewScanner(r)

	var links []link.Link
	record := 0

	for scanner.Scan() {
		record++

		var l link.Link

		if err := json.Unmarshal(scanner.Bytes(), &l); err != nil {
			return nil, fmt.Errorf("record %d: %w", record, err)
		}

		links = append(links, l)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read links: %w", err)
	}

	return links, nil
}

func WriteLinks(w io.Writer, links []link.Link) error {
	encoder := json.NewEncoder(w)

	for _, l := range links {
		if err := encoder.Encode(l); err != nil {
			return fmt.Errorf("write link: %w", err)
		}
	}

	return nil
}

func WriteCSV(w io.Writer, links []link.Link) error {
	writer := csv.NewWriter(w)

	header := []string{
		"id",
		"code",
		"target",
		"owner_id",
		"created_at",
		"expires_at",
	}

	if err := writer.Write(header); err != nil {
		return fmt.Errorf("write CSV header: %w", err)
	}

	for _, l := range links {
		expiresAt := ""

		if l.ExpiresAt.IsZero() {
			expiresAt = l.ExpiresAt.UTC().Format(time.RFC3339)
		}

		record := []string{
			strconv.FormatUint(l.ID, 10),
			l.Code,
			l.Target,
			l.OwnerID,
			l.CreatedAt.UTC().Format(time.RFC3339),
			expiresAt,
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("write CSV record: %w", err)
		}
	}

	writer.Flush()

	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush CSV: %w", err)
	}

	return nil
}

func LoadFile(path string) ([]link.Link, error) {
	file, err := os.Open(path)

	if err != nil {
		return nil, fmt.Errorf("load file %q: %w", path, err)
	}
	defer file.Close()

	links, err := ReadLinks(file)

	if err != nil {
		return nil, fmt.Errorf("load file %q: %w", path, err)
	}

	return links, nil
}

func SaveFile(path string, links []link.Link) error {
	file, err := os.Create(path)

	if err != nil {
		return fmt.Errorf("save file %q: %w", path, err)
	}

	writeErr := WriteLinks(file, links)
	closeErr := file.Close()

	if writeErr != nil {
		writeErr = fmt.Errorf("save file %q: %w", path, writeErr)
	}

	if closeErr != nil {
		closeErr = fmt.Errorf("close file %q: %w", path, closeErr)
	}

	return errors.Join(writeErr, closeErr)
}
