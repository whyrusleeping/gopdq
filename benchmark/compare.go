package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/why/gopdq"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run compare.go <image_file_or_directory>")
		fmt.Println("       make compare IMAGE=<path>")
		os.Exit(1)
	}

	path := os.Args[1]
	
	// Check if path is file or directory
	info, err := os.Stat(path)
	if err != nil {
		log.Fatalf("Error accessing path: %v", err)
	}

	var imageFiles []string
	if info.IsDir() {
		imageFiles, err = findImageFiles(path)
		if err != nil {
			log.Fatalf("Error finding images: %v", err)
		}
		if len(imageFiles) == 0 {
			log.Fatalf("No image files found in directory: %s", path)
		}
	} else {
		imageFiles = []string{path}
	}

	fmt.Printf("Comparing PDQ implementations on %d image(s)\n", len(imageFiles))
	fmt.Println(strings.Repeat("=", 80))

	hasher := gopdq.NewPdqHasher()
	var totalBitDifferences int
	var successfulComparisons int

	for _, imagePath := range imageFiles {
		// Get Facebook's hash
		facebookHash, err := getFacebookHash(imagePath)
		if err != nil {
			fmt.Printf("❌ %s: Failed to get Facebook hash: %v\n", filepath.Base(imagePath), err)
			continue
		}

		// Get our Go hash
		result, err := hasher.FromFile(imagePath)
		if err != nil {
			fmt.Printf("❌ %s: Failed to get Go hash: %v\n", filepath.Base(imagePath), err)
			continue
		}

		goHash := result.Hash.String()
		
		// Compare hashes
		bitDifferences := countBitDifferences(facebookHash, goHash)
		totalBitDifferences += bitDifferences
		successfulComparisons++

		// Display results
		if bitDifferences == 0 {
			fmt.Printf("✅ %s: EXACT MATCH\n", filepath.Base(imagePath))
		} else {
			fmt.Printf("⚠️  %s: %d bit differences (%.1f%%)\n", 
				filepath.Base(imagePath), 
				bitDifferences, 
				float64(bitDifferences)/256.0*100)
		}
		
		fmt.Printf("   Facebook: %s\n", facebookHash)
		fmt.Printf("   Go:       %s\n", goHash)
		fmt.Printf("   Quality:  %d\n", result.Quality)
		
		if bitDifferences > 0 && len(imageFiles) == 1 {
			// Show detailed bit analysis for single image
			showBitAnalysis(facebookHash, goHash)
		}
		fmt.Println()
	}

	// Summary
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("SUMMARY")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Images processed: %d\n", successfulComparisons)
	if successfulComparisons > 0 {
		avgBitDifferences := float64(totalBitDifferences) / float64(successfulComparisons)
		fmt.Printf("Average bit differences: %.1f bits (%.2f%%)\n", 
			avgBitDifferences, 
			avgBitDifferences/256.0*100)
		
		if totalBitDifferences == 0 {
			fmt.Println("🎉 All hashes match exactly!")
		} else {
			fmt.Printf("📊 Total bit differences: %d across all images\n", totalBitDifferences)
		}
	}
}

func getFacebookHash(imagePath string) (string, error) {
	// Check if Facebook's pdq-photo-hasher exists
	facebookHasher := "facebook-pdq/pdq/cpp/pdq-photo-hasher"
	if _, err := os.Stat(facebookHasher); os.IsNotExist(err) {
		return "", fmt.Errorf("Facebook's pdq-photo-hasher not found at %s. Run 'make setup' first", facebookHasher)
	}

	// Run Facebook's hasher
	cmd := exec.Command(facebookHasher, imagePath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run Facebook hasher: %v", err)
	}

	// Parse output: "hash,quality,filename"
	parts := strings.Split(strings.TrimSpace(string(output)), ",")
	if len(parts) < 1 {
		return "", fmt.Errorf("unexpected output format from Facebook hasher")
	}

	return parts[0], nil
}

func countBitDifferences(hash1, hash2 string) int {
	if len(hash1) != len(hash2) {
		return -1 // Invalid comparison
	}

	differences := 0
	for i := 0; i < len(hash1); i++ {
		if hash1[i] != hash2[i] {
			// Convert hex characters to values and count differing bits
			val1 := hexCharToInt(hash1[i])
			val2 := hexCharToInt(hash2[i])
			xor := val1 ^ val2
			
			// Count set bits in XOR result
			for xor > 0 {
				if xor&1 == 1 {
					differences++
				}
				xor >>= 1
			}
		}
	}
	return differences
}

func hexCharToInt(c byte) int {
	if c >= '0' && c <= '9' {
		return int(c - '0')
	}
	if c >= 'a' && c <= 'f' {
		return int(c - 'a' + 10)
	}
	if c >= 'A' && c <= 'F' {
		return int(c - 'A' + 10)
	}
	return 0
}

func showBitAnalysis(hash1, hash2 string) {
	fmt.Println("   Bit Analysis:")
	
	differentPositions := []int{}
	for i := 0; i < len(hash1); i++ {
		if hash1[i] != hash2[i] {
			differentPositions = append(differentPositions, i)
		}
	}
	
	fmt.Printf("   Different hex positions: %v\n", differentPositions)
	
	// Show first few differences in detail
	maxShow := 5
	if len(differentPositions) > maxShow {
		fmt.Printf("   (showing first %d of %d differences)\n", maxShow, len(differentPositions))
	}
	
	for i, pos := range differentPositions {
		if i >= maxShow {
			break
		}
		val1 := hexCharToInt(hash1[pos])
		val2 := hexCharToInt(hash2[pos])
		fmt.Printf("     Pos %d: %c (%04b) vs %c (%04b) - XOR: %04b\n", 
			pos, hash1[pos], val1, hash2[pos], val2, val1^val2)
	}
}

func findImageFiles(dirPath string) ([]string, error) {
	var imageFiles []string
	
	imageExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".bmp":  true,
		".tiff": true,
		".webp": true,
	}

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if imageExts[ext] {
			imageFiles = append(imageFiles, path)
		}

		return nil
	})

	return imageFiles, err
}