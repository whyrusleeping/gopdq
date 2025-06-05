package main

import (
	"fmt"
	"log"
	"os"

	"github.com/why/gopdq"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <image-file>")
		os.Exit(1)
	}

	imagePath := os.Args[1]

	// Create a new hasher
	hasher := gopdq.NewPdqHasher()

	// Compute hash from file
	result, err := hasher.FromFile(imagePath)
	if err != nil {
		log.Fatalf("Failed to compute hash: %v", err)
	}

	// Print results
	fmt.Printf("PDQ Hash: %s\n", result.Hash.String())
	fmt.Printf("Quality: %d\n", result.Quality)
	fmt.Printf("Read time: %.3f seconds\n", result.Stats.ReadSeconds)
	fmt.Printf("Hash time: %.3f seconds\n", result.Stats.HashSeconds)
	fmt.Printf("Image size: %d pixels\n", result.Stats.NumPixels)
	
	// Demonstrate some hash operations
	fmt.Println("\nHash operations:")
	fmt.Printf("Hamming norm (number of 1 bits): %d\n", result.Hash.HammingNorm())
	
	// Create a slightly modified hash for comparison
	fuzzedHash := result.Hash.Fuzz(5) // Flip 5 random bits
	fmt.Printf("\nFuzzed hash: %s\n", fuzzedHash.String())
	fmt.Printf("Hamming distance from original: %d\n", result.Hash.HammingDistance(fuzzedHash))
	
	// Test hex string conversion
	hexString := result.Hash.ToHexString()
	parsedHash, err := gopdq.FromHexString(hexString)
	if err != nil {
		log.Printf("Failed to parse hex string: %v", err)
	} else {
		fmt.Printf("\nHash roundtrip successful: %v\n", result.Hash.Equal(parsedHash))
	}
}