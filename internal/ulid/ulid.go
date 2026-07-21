// Package ulid generates ULIDs: 26-character, Crockford-base32, millisecond
// timestamp + 80 random bits. Lexicographic order is creation order, which is
// the property the event log's whole design rests on — a directory listing of
// ULID-named files IS the ordered log.
//
// Implemented here rather than imported: it is ~60 lines, and the zero-
// dependency property of the module is worth more than a library.
package ulid

import (
	"crypto/rand"
	"fmt"
	"time"
)

// Crockford base32: no I, L, O, U — unambiguous when read by humans off a
// directory listing, which is how these get read.
const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// New returns a fresh ULID for the current time.
func New() string {
	return At(time.Now())
}

// At returns a ULID for the given time. Split out for tests, which need
// deterministic timestamps to assert ordering.
func At(t time.Time) string {
	var b [26]byte

	// 48-bit millisecond timestamp, 10 base32 chars, big-endian so that
	// string order equals time order.
	ms := uint64(t.UnixMilli())
	for i := 9; i >= 0; i-- {
		b[i] = alphabet[ms&31]
		ms >>= 5
	}

	// 80 random bits, 16 base32 chars. crypto/rand: a collision here is a
	// corrupted event log, so this is not the place for math/rand.
	var r [10]byte
	if _, err := rand.Read(r[:]); err != nil {
		// crypto/rand failing means the platform is broken in a way no
		// fallback should paper over.
		panic(fmt.Sprintf("ulid: crypto/rand failed: %v", err))
	}
	// 10 bytes = 80 bits → 16 groups of 5 bits.
	var acc uint64
	bits := 0
	pos := 10
	for _, by := range r {
		acc = acc<<8 | uint64(by)
		bits += 8
		for bits >= 5 {
			bits -= 5
			b[pos] = alphabet[(acc>>uint(bits))&31]
			pos++
		}
	}
	return string(b[:])
}

// Valid reports whether s looks like a ULID: 26 chars from the alphabet.
func Valid(s string) bool {
	if len(s) != 26 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
		case c >= 'A' && c <= 'Z' && c != 'I' && c != 'L' && c != 'O' && c != 'U':
		default:
			return false
		}
	}
	return true
}
