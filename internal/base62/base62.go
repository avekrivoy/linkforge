package base62

import (
	"slices"
	"strings"
)

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// divider returns quotient and remainder
func divider(dividend, divisor uint64) (uint64, uint64) {
	return dividend / divisor, dividend % divisor
}

// Encode renders a number with no padding.
func Encode(v uint64) string {
	var encoded []string
	for {
		quotient, remainder := divider(v, 62)

		encoded = append(encoded, string(alphabet[remainder]))
		if quotient == 0 {
			break
		}
		v = quotient
	}
	slices.Reverse(encoded)

	return strings.Join(encoded, "")
}
