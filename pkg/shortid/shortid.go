package shortid

import (
	"fmt"
	"io"

	"github.com/avekrivoy/linkforge/internal/base62"
)

const (
	randomBits  = 19
	counterBits = 16
)

type CodeGenerator struct {
	random  io.Reader
	counter uint16
}

var _ Generator = (*CodeGenerator)(nil)

func NewCodeGenerator(random io.Reader) *CodeGenerator {
	return &CodeGenerator{
		random: random,
	}
}

func (g *CodeGenerator) Next() (string, error) {
	var buf [3]byte

	if _, err := io.ReadFull(g.random, buf[:]); err != nil {
		return "", fmt.Errorf("shortid: read random bytes: %w", err)
	}

	randomValue := uint32(buf[0])<<16 |
		uint32(buf[1])<<8 |
		uint32(buf[2])

	r := randomValue & ((1 << randomBits) - 1)
	value := (r << counterBits) | uint32(g.counter)
	g.counter++

	return base62.EncodeWidth(uint64(value), 6), nil
}
