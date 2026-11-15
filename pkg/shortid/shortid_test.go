package shortid

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/avekrivoy/linkforge/internal/base62"
)

type repeatReader struct {
	value byte
}

func (r *repeatReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = r.value
	}

	return len(p), nil
}

type failOnceReader struct {
	failed bool
	reader *bytes.Reader
	err    error
}

func (r *failOnceReader) Read(p []byte) (int, error) {
	if !r.failed {
		r.failed = true
		return 0, r.err
	}

	return r.reader.Read(p)
}

func TestNewCodeGenerator(t *testing.T) {
	g := NewCodeGenerator(bytes.NewReader([]byte{0, 0, 0}))

	if g == nil {
		t.Fatal("NewCodeGenerator() returned nil")
	}
}

func TestCodeGeneratorReferenceValues(t *testing.T) {
	t.Run("random bytes 00 00 01", func(t *testing.T) {
		data := bytes.Repeat([]byte{0x00, 0x00, 0x01}, 3)
		g := NewCodeGenerator(bytes.NewReader(data))

		want := []string{
			"000H32",
			"000H33",
			"000H34",
		}

		for i, expected := range want {
			got, err := g.Next()
			if err != nil {
				t.Fatalf("Next() call %d error = %v", i, err)
			}

			if got != expected {
				t.Errorf("Next() call %d = %q, want %q", i, got, expected)
			}
		}
	})

	t.Run("random bytes FF FF FF", func(t *testing.T) {
		g := NewCodeGenerator(bytes.NewReader([]byte{
			0xFF, 0xFF, 0xFF,
		}))

		got, err := g.Next()
		if err != nil {
			t.Fatalf("Next() error = %v", err)
		}

		const want = "bVJgVU"

		if got != want {
			t.Errorf("Next() = %q, want %q", got, want)
		}
	})

	t.Run("random bytes 00 00 00", func(t *testing.T) {
		g := NewCodeGenerator(bytes.NewReader([]byte{
			0x00, 0x00, 0x00,
			0x00, 0x00, 0x00,
		}))

		want := []string{
			"000000",
			"000001",
		}

		for i, expected := range want {
			got, err := g.Next()
			if err != nil {
				t.Fatalf("Next() call %d error = %v", i, err)
			}

			if got != expected {
				t.Errorf("Next() call %d = %q, want %q", i, got, expected)
			}
		}
	})
}

func TestCodeGeneratorReadsExactlyThreeBytes(t *testing.T) {
	reader := bytes.NewReader([]byte{
		0x00, 0x00, 0x01,
		0xAA,
	})

	g := NewCodeGenerator(reader)

	_, err := g.Next()
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}

	if got := reader.Len(); got != 1 {
		t.Errorf("remaining bytes = %d, want 1", got)
	}
}

func TestCodeGeneratorUsesLow19Bits(t *testing.T) {
	g := NewCodeGenerator(bytes.NewReader([]byte{
		0x08, 0x00, 0x01,
	}))

	got, err := g.Next()
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}

	want := base62.EncodeWidth(1<<16, 6)

	if got != want {
		t.Errorf("Next() = %q, want %q", got, want)
	}
}

func TestCodeGeneratorError(t *testing.T) {
	t.Run("reader returns error", func(t *testing.T) {
		readerErr := errors.New("random source failed")

		reader := &failOnceReader{
			reader: bytes.NewReader([]byte{0, 0, 0}),
			err:    readerErr,
		}

		g := NewCodeGenerator(reader)

		got, err := g.Next()

		if got != "" {
			t.Errorf("Next() code = %q, want empty string", got)
		}

		if !errors.Is(err, readerErr) {
			t.Errorf("Next() error = %v, want wrapped reader error", err)
		}
	})

	t.Run("counter does not change after error", func(t *testing.T) {
		readerErr := errors.New("temporary error")

		reader := &failOnceReader{
			reader: bytes.NewReader([]byte{
				0x00, 0x00, 0x00,
			}),
			err: readerErr,
		}

		g := NewCodeGenerator(reader)

		_, err := g.Next()
		if !errors.Is(err, readerErr) {
			t.Fatalf("Next() error = %v, want wrapped reader error", err)
		}

		got, err := g.Next()
		if err != nil {
			t.Fatalf("second Next() error = %v", err)
		}

		if got != "000000" {
			t.Errorf("second Next() = %q, want %q", got, "000000")
		}
	})

	t.Run("not enough bytes", func(t *testing.T) {
		g := NewCodeGenerator(bytes.NewReader([]byte{
			0x00, 0x00,
		}))

		got, err := g.Next()

		if got != "" {
			t.Errorf("Next() code = %q, want empty string", got)
		}

		if !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Errorf("Next() error = %v, want io.ErrUnexpectedEOF", err)
		}
	})

	t.Run("empty reader", func(t *testing.T) {
		g := NewCodeGenerator(bytes.NewReader(nil))

		got, err := g.Next()

		if got != "" {
			t.Errorf("Next() code = %q, want empty string", got)
		}

		if !errors.Is(err, io.EOF) {
			t.Errorf("Next() error = %v, want io.EOF", err)
		}
	})
}

func TestCodeGeneratorCounterWrap(t *testing.T) {
	g := NewCodeGenerator(&repeatReader{value: 0})

	for i := 0; i < 1<<16; i++ {
		_, err := g.Next()
		if err != nil {
			t.Fatalf("Next() call %d error = %v", i, err)
		}
	}

	got, err := g.Next()
	if err != nil {
		t.Fatalf("Next() after wrap error = %v", err)
	}

	if got != "000000" {
		t.Errorf("Next() after wrap = %q, want %q", got, "000000")
	}
}

func TestCodeGeneratorCodeWidth(t *testing.T) {
	g := NewCodeGenerator(&repeatReader{value: 0xFF})

	for i := 0; i < 100; i++ {
		code, err := g.Next()
		if err != nil {
			t.Fatalf("Next() call %d error = %v", i, err)
		}

		if len(code) != 6 {
			t.Errorf("code %q has length %d, want 6", code, len(code))
		}
	}
}

func TestCodeGeneratorCounterIncrements(t *testing.T) {
	g := NewCodeGenerator(&repeatReader{value: 0})

	first, err := g.Next()
	if err != nil {
		t.Fatal(err)
	}

	second, err := g.Next()
	if err != nil {
		t.Fatal(err)
	}

	third, err := g.Next()
	if err != nil {
		t.Fatal(err)
	}

	if first != "000000" {
		t.Errorf("first code = %q, want 000000", first)
	}

	if second != "000001" {
		t.Errorf("second code = %q, want 000001", second)
	}

	if third != "000002" {
		t.Errorf("third code = %q, want 000002", third)
	}
}
