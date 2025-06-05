package gopdq

import (
	"testing"
)

func TestNewPdqHash256(t *testing.T) {
	hash := NewPdqHash256()
	if hash == nil {
		t.Fatal("NewPdqHash256 returned nil")
	}
	
	// Check all words are zero
	for i, w := range hash.w {
		if w != 0 {
			t.Errorf("Word %d is not zero: %d", i, w)
		}
	}
}

func TestPdqHash256String(t *testing.T) {
	hash := NewPdqHash256()
	
	// Empty hash should be all zeros
	expected := "0000000000000000000000000000000000000000000000000000000000000000"
	if hash.String() != expected {
		t.Errorf("Expected %s, got %s", expected, hash.String())
	}
	
	// Set some bits
	hash.SetBit(0)
	hash.SetBit(255)
	str := hash.String()
	if len(str) != 64 {
		t.Errorf("Hash string length should be 64, got %d", len(str))
	}
}

func TestPdqHash256SetBit(t *testing.T) {
	hash := NewPdqHash256()
	
	// Set bit 0
	hash.SetBit(0)
	if hash.w[0] != 1 {
		t.Errorf("Bit 0 not set correctly")
	}
	
	// Set bit 15
	hash.SetBit(15)
	if hash.w[0] != 0x8001 {
		t.Errorf("Bit 15 not set correctly: %x", hash.w[0])
	}
	
	// Set bit 16 (should be in word 1)
	hash.SetBit(16)
	if hash.w[1] != 1 {
		t.Errorf("Bit 16 not set correctly")
	}
}

func TestPdqHash256FlipBit(t *testing.T) {
	hash := NewPdqHash256()
	
	// Flip bit 0 (0 -> 1)
	hash.FlipBit(0)
	if hash.w[0] != 1 {
		t.Errorf("Bit 0 not flipped correctly")
	}
	
	// Flip bit 0 again (1 -> 0)
	hash.FlipBit(0)
	if hash.w[0] != 0 {
		t.Errorf("Bit 0 not flipped back correctly")
	}
}

func TestPdqHash256HammingNorm(t *testing.T) {
	hash := NewPdqHash256()
	
	// Empty hash should have norm 0
	if hash.HammingNorm() != 0 {
		t.Errorf("Empty hash should have norm 0")
	}
	
	// Set some bits
	hash.SetBit(0)
	hash.SetBit(1)
	hash.SetBit(2)
	if hash.HammingNorm() != 3 {
		t.Errorf("Expected norm 3, got %d", hash.HammingNorm())
	}
}

func TestPdqHash256HammingDistance(t *testing.T) {
	hash1 := NewPdqHash256()
	hash2 := NewPdqHash256()
	
	// Same hashes should have distance 0
	if hash1.HammingDistance(hash2) != 0 {
		t.Errorf("Same hashes should have distance 0")
	}
	
	// Different by one bit
	hash1.SetBit(0)
	if hash1.HammingDistance(hash2) != 1 {
		t.Errorf("Expected distance 1, got %d", hash1.HammingDistance(hash2))
	}
	
	// Different by multiple bits
	hash1.SetBit(10)
	hash1.SetBit(20)
	if hash1.HammingDistance(hash2) != 3 {
		t.Errorf("Expected distance 3, got %d", hash1.HammingDistance(hash2))
	}
}

func TestPdqHash256Equal(t *testing.T) {
	hash1 := NewPdqHash256()
	hash2 := NewPdqHash256()
	
	if !hash1.Equal(hash2) {
		t.Error("Empty hashes should be equal")
	}
	
	hash1.SetBit(0)
	if hash1.Equal(hash2) {
		t.Error("Different hashes should not be equal")
	}
	
	hash2.SetBit(0)
	if !hash1.Equal(hash2) {
		t.Error("Same hashes should be equal")
	}
}

func TestPdqHash256BitwiseOperations(t *testing.T) {
	hash1 := NewPdqHash256()
	hash2 := NewPdqHash256()
	
	// Set different bits
	hash1.SetBit(0)
	hash1.SetBit(2)
	hash2.SetBit(1)
	hash2.SetBit(2)
	
	// Test XOR
	xor := hash1.Xor(hash2)
	if xor.w[0] != 3 { // bits 0 and 1 should be set
		t.Errorf("XOR failed: expected 3, got %d", xor.w[0])
	}
	
	// Test AND
	and := hash1.And(hash2)
	if and.w[0] != 4 { // only bit 2 should be set
		t.Errorf("AND failed: expected 4, got %d", and.w[0])
	}
	
	// Test OR
	or := hash1.Or(hash2)
	if or.w[0] != 7 { // bits 0, 1, and 2 should be set
		t.Errorf("OR failed: expected 7, got %d", or.w[0])
	}
}

func TestPdqHash256FromHexString(t *testing.T) {
	// Test valid hex string
	hexStr := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	hash, err := FromHexString(hexStr)
	if err != nil {
		t.Fatalf("Failed to parse valid hex string: %v", err)
	}
	
	// Verify round trip
	if hash.ToHexString() != hexStr {
		t.Errorf("Hex string round trip failed")
	}
	
	// Test invalid length
	_, err = FromHexString("0123")
	if err == nil {
		t.Error("Should fail with invalid length")
	}
	
	// Test invalid characters
	_, err = FromHexString("g123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if err == nil {
		t.Error("Should fail with invalid hex characters")
	}
}

func TestPdqHash256Clone(t *testing.T) {
	hash1 := NewPdqHash256()
	hash1.SetBit(0)
	hash1.SetBit(100)
	
	hash2 := hash1.Clone()
	
	// Should be equal
	if !hash1.Equal(hash2) {
		t.Error("Cloned hash should be equal")
	}
	
	// Modifying clone should not affect original
	hash2.SetBit(200)
	if hash1.Equal(hash2) {
		t.Error("Modifying clone should not affect original")
	}
}

func TestPdqHash256Fuzz(t *testing.T) {
	hash := NewPdqHash256()
	hash.SetAll() // Set all bits to 1
	
	// Fuzz with 10 bit flips
	fuzzed := hash.Fuzz(10)
	
	// Distance should be approximately 10 (some bits might flip twice)
	distance := hash.HammingDistance(fuzzed)
	if distance == 0 || distance > 10 {
		t.Errorf("Unexpected distance after fuzzing: %d", distance)
	}
}