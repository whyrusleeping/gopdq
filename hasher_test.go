package gopdq

import (
	"bytes"
	"fmt"
	_ "image/jpeg"
	"os"
	"testing"
)

func TestKnownImage(t *testing.T) {
	hasher := NewPdqHasher()

	res, err := hasher.FromFile("cat.jpg")
	if err != nil {
		t.Fatal(err)
	}

	exp := "06704e1dd910f233c0e6df833130b0ff99e36701383d333ac7c6078fe736dccc"
	got := res.Hash.String()
	if got != exp {
		for i := 0; i < len(exp); i++ {
			if got[i] != exp[i] {
				fmt.Printf("mismatch at index %d: %s\n", i, got[i:i+1])
			}
		}
		t.Fatal("hash mismatch: ", res.Hash.String(), exp)
	}
}

func BenchmarkHashing(b *testing.B) {
	data, err := os.ReadFile("cat.jpg")
	if err != nil {
		b.Fatal(err)
	}

	img, err := DecodeJpeg(bytes.NewReader(data))
	if err != nil {
		b.Fatal(err)
	}

	hasher := NewPdqHasher()

	for i := 0; i < b.N; i++ {
		_, err := hasher.HashImage(img)
		if err != nil {
			b.Fatal(err)
		}
	}
}
