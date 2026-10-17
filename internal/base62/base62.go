package base62

import (
	"errors"
	"math"
	"slices"
	"strings"
)

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var (
	ErrEmpty            = errors.New("base62: empty code")
	ErrInvalidCharacter = errors.New("base62: invalid character")
	ErrOverflow         = errors.New("base62: code is too long")
)

// Encode renders a number with no padding.
func Encode(v uint64) string {
	var encoded []string
	for {
		quotient, remainder := v/62, v%62
		encoded = append(encoded, string(alphabet[remainder]))
		if quotient == 0 {
			break
		}
		v = quotient
	}
	slices.Reverse(encoded)

	return strings.Join(encoded, "")
}

// EncodeWidth does the same but left-pads with zeroes to at least width characters. The value itself is never truncated.
func EncodeWidth(v uint64, width int) string {
	encoded := Encode(v)

	if width > len(encoded) {
		fillers := strings.Repeat("0", width-len(encoded))
		return fillers + encoded
	}

	return encoded
}

// EncodeRecursive returns the same result as Encode
func EncodeRecursive(v uint64) string {
	var encoded strings.Builder

	quotient, remainder := v/62, v%62
	encoded.WriteString(string(alphabet[remainder]))
	if quotient == 0 {
		return encoded.String()
	}

	return EncodeRecursive(quotient) + encoded.String()
}

// Decode parses a code back into a number.
func Decode(s string) (uint64, error) {
	if len(s) == 0 {
		return 0, ErrEmpty
	}

	if len(s) > 11 {
		return 0, ErrOverflow
	}

	trimmed := strings.TrimLeft(s, "0")

	var decoded uint64

	for i := len(trimmed); i > 0; i-- {
		if strings.ContainsAny(alphabet, string(trimmed[i-1])) {
			power := len(trimmed) - i
			positionValue := strings.Index(alphabet, string(trimmed[i-1]))

			decodedChar := uint64(float64(positionValue) * math.Pow(62, float64(power)))

			if math.MaxUint64-decoded < decodedChar {
				return 0, ErrOverflow
			}

			decoded += decodedChar
			continue
		}
		return 0, ErrInvalidCharacter
	}

	return decoded, nil
}
