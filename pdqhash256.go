package gopdq

import (
	"fmt"
	"math/bits"
	"math/rand"
	"strconv"
	"strings"
)

const (
	HASH256NUMSLOTS          = 16
	HASH256_HEX_NUM_NYBBLES = 4 * HASH256NUMSLOTS
)

// PdqHash256 represents a 256-bit PDQ hash
type PdqHash256 struct {
	w   [HASH256NUMSLOTS]int
	rnd *rand.Rand
}

// NewPdqHash256 creates a new PdqHash256 instance
func NewPdqHash256() *PdqHash256 {
	return &PdqHash256{
		w:   [HASH256NUMSLOTS]int{},
		rnd: rand.New(rand.NewSource(rand.Int63())),
	}
}

// GetNumWords returns the number of words in the hash
func GetNumWords() int {
	return HASH256NUMSLOTS
}

// String returns the hexadecimal string representation of the hash
func (h *PdqHash256) String() string {
	var sb strings.Builder
	for i := HASH256NUMSLOTS - 1; i >= 0; i-- {
		sb.WriteString(fmt.Sprintf("%04x", h.w[i]&0xFFFF))
	}
	return sb.String()
}

// Clear sets all bits to zero
func (h *PdqHash256) Clear() {
	for i := 0; i < HASH256NUMSLOTS; i++ {
		h.w[i] = 0
	}
}

// SetAll sets all bits to one
func (h *PdqHash256) SetAll() {
	for i := 0; i < HASH256NUMSLOTS; i++ {
		h.w[i] = 0xFFFF
	}
}

// HammingNorm returns the number of set bits
func (h *PdqHash256) HammingNorm() int {
	n := 0
	for i := 0; i < HASH256NUMSLOTS; i++ {
		n += hammingNorm16(h.w[i])
	}
	return n
}

// HammingDistance calculates the Hamming distance between two hashes
func (h *PdqHash256) HammingDistance(other *PdqHash256) int {
	n := 0
	for i := 0; i < HASH256NUMSLOTS; i++ {
		n += hammingNorm16(h.w[i] ^ other.w[i])
	}
	return n
}

// HammingDistanceLE checks if Hamming distance is less than or equal to d
func (h *PdqHash256) HammingDistanceLE(that *PdqHash256, d int) bool {
	e := 0
	for i := 0; i < HASH256NUMSLOTS; i++ {
		e += hammingNorm16(h.w[i] ^ that.w[i])
		if e > d {
			return false
		}
	}
	return true
}

// SetBit sets the bit at position k
func (h *PdqHash256) SetBit(k int) {
	h.w[(k&255)>>4] |= 1 << (k & 15)
}

// FlipBit flips the bit at position k
func (h *PdqHash256) FlipBit(k int) {
	h.w[(k&255)>>4] ^= 1 << (k & 15)
}

// Xor performs bitwise XOR with another hash
func (h *PdqHash256) Xor(other *PdqHash256) *PdqHash256 {
	rv := NewPdqHash256()
	for i := 0; i < HASH256NUMSLOTS; i++ {
		rv.w[i] = h.w[i] ^ other.w[i]
	}
	return rv
}

// And performs bitwise AND with another hash
func (h *PdqHash256) And(other *PdqHash256) *PdqHash256 {
	rv := NewPdqHash256()
	for i := 0; i < HASH256NUMSLOTS; i++ {
		rv.w[i] = h.w[i] & other.w[i]
	}
	return rv
}

// Or performs bitwise OR with another hash
func (h *PdqHash256) Or(other *PdqHash256) *PdqHash256 {
	rv := NewPdqHash256()
	for i := 0; i < HASH256NUMSLOTS; i++ {
		rv.w[i] = h.w[i] | other.w[i]
	}
	return rv
}

// BitwiseNOT performs bitwise NOT operation
func (h *PdqHash256) BitwiseNOT() *PdqHash256 {
	rv := NewPdqHash256()
	for i := 0; i < HASH256NUMSLOTS; i++ {
		rv.w[i] = ^h.w[i] & 0xFFFF
	}
	return rv
}

// Equal checks if two hashes are equal
func (h *PdqHash256) Equal(other *PdqHash256) bool {
	for i := 0; i < HASH256NUMSLOTS; i++ {
		if h.w[i] != other.w[i] {
			return false
		}
	}
	return true
}

// Less checks if this hash is less than another
func (h *PdqHash256) Less(other *PdqHash256) bool {
	for i := 0; i < HASH256NUMSLOTS; i++ {
		if h.w[i] < other.w[i] {
			return true
		} else if h.w[i] > other.w[i] {
			return false
		}
	}
	return false
}

// Greater checks if this hash is greater than another
func (h *PdqHash256) Greater(other *PdqHash256) bool {
	for i := 0; i < HASH256NUMSLOTS; i++ {
		if h.w[i] > other.w[i] {
			return true
		} else if h.w[i] < other.w[i] {
			return false
		}
	}
	return false
}

// DumpBits returns a string representation of the bits
func (h *PdqHash256) DumpBits() string {
	var lines []string
	for i := HASH256NUMSLOTS - 1; i >= 0; i-- {
		word := h.w[i] & 0xFFFF
		var bits []string
		for j := 15; j >= 0; j-- {
			if (word & (1 << j)) != 0 {
				bits = append(bits, "1")
			} else {
				bits = append(bits, "0")
			}
		}
		lines = append(lines, strings.Join(bits, " "))
	}
	return strings.Join(lines, "\n")
}

// ToBits returns the bits as a slice of bytes
func (h *PdqHash256) ToBits() []byte {
	var bits []byte
	for i := HASH256NUMSLOTS - 1; i >= 0; i-- {
		word := h.w[i] & 0xFFFF
		for j := 15; j >= 0; j-- {
			if (word & (1 << j)) != 0 {
				bits = append(bits, 1)
			} else {
				bits = append(bits, 0)
			}
		}
	}
	return bits
}

// DumpBitsAcross returns a string representation of bits in one line
func (h *PdqHash256) DumpBitsAcross() string {
	var str []string
	for i := HASH256NUMSLOTS - 1; i >= 0; i-- {
		word := h.w[i] & 0xFFFF
		for j := 15; j >= 0; j-- {
			if (word & (1 << j)) != 0 {
				str = append(str, "1")
			} else {
				str = append(str, "0")
			}
		}
	}
	return strings.Join(str, " ")
}

// DumpWords returns a string representation of the words
func (h *PdqHash256) DumpWords() string {
	var words []string
	for i := HASH256NUMSLOTS - 1; i >= 0; i-- {
		words = append(words, strconv.Itoa(h.w[i]))
	}
	return strings.Join(words, ",")
}

// Words returns a copy of the internal words array
func (h *PdqHash256) Words() []int {
	words := make([]int, HASH256NUMSLOTS)
	copy(words, h.w[:])
	return words
}

// Clone creates a deep copy of the hash
func (h *PdqHash256) Clone() *PdqHash256 {
	rv := NewPdqHash256()
	copy(rv.w[:], h.w[:])
	rv.rnd = h.rnd
	return rv
}

// Fuzz flips some number of bits randomly, with replacement
func (h *PdqHash256) Fuzz(numErrorBits int) *PdqHash256 {
	rv := h.Clone()
	for i := 0; i < numErrorBits; i++ {
		rv.FlipBit(rv.rnd.Intn(256))
	}
	return rv
}

// ToHexString returns the hexadecimal string representation
func (h *PdqHash256) ToHexString() string {
	return h.String()
}

// FromHexString creates a PdqHash256 from a hexadecimal string
func FromHexString(hexString string) (*PdqHash256, error) {
	if len(hexString) != HASH256_HEX_NUM_NYBBLES {
		return nil, fmt.Errorf("incorrect hex length for pdq hash: expected %d, got %d", HASH256_HEX_NUM_NYBBLES, len(hexString))
	}

	rv := NewPdqHash256()
	i := HASH256NUMSLOTS

	for x := 0; x < len(hexString); x += 4 {
		i--
		val, err := strconv.ParseInt(hexString[x:x+4], 16, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse hex string: %w", err)
		}
		rv.w[i] = int(val)
	}

	return rv, nil
}

// hammingNorm16 counts the number of set bits in a 16-bit value
func hammingNorm16(v int) int {
	return bits.OnesCount16(uint16(v & 0xFFFF))
}