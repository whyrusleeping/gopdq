package gopdq

import (
	"bytes"
	"image"
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
	if res.Hash.String() != exp {
		t.Fatal("hash mismatch: ", res.Hash.String(), exp)
	}
}

func BenchmarkHashing(b *testing.B) {
	data, err := os.ReadFile("cat.jpg")
	if err != nil {
		b.Fatal(err)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
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
