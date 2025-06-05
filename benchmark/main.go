package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/whyrusleeping/gopdq"
)

type BenchmarkResult struct {
	ImageName    string
	ImageSize    string
	Pixels       int
	HashesPerSec float64
	AvgHashTime  float64
	Quality      int
	Hash         string
}

func main() {
	// Parse command line arguments
	var (
		verbose   = flag.Bool("v", false, "Verbose output")
		numHashes = flag.Int("n", 0, "Total number of hashes to generate (0 means all images)")
		help      = flag.Bool("h", false, "Show help")
	)
	flag.Parse()

	if *help {
		usage()
		return
	}

	fmt.Println("PDQ Hash Benchmark")
	fmt.Printf("Running on %d CPU cores\n", runtime.NumCPU())
	fmt.Println("==================")

	// Check if directory argument is provided
	args := flag.Args()
	if len(args) > 0 {
		// Directory mode - benchmark real images
		dirPath := args[0]
		benchmarkDirectory(dirPath, *verbose, *numHashes)
	} else {
		// Synthetic benchmark mode
		benchmarkSynthetic(*verbose)
	}
}

func usage() {
	fmt.Println("Usage: go run benchmark/main.go [options] [folder_path]")
	fmt.Println("Options:")
	fmt.Println("  -v               Verbose output")
	fmt.Println("  -n N             Total number of hashes to generate (default: 0, all images)")
	fmt.Println("  -h               Show this help")
	fmt.Println("")
	fmt.Println("If folder_path is provided, benchmarks real images from that directory.")
	fmt.Println("Otherwise, generates synthetic test images for benchmarking.")
}

func benchmarkSynthetic(verbose bool) {
	// Create test images directory
	testDir := "benchmark_images"
	if err := os.MkdirAll(testDir, 0755); err != nil {
		log.Fatalf("Failed to create test directory: %v", err)
	}

	// Generate test images
	fmt.Println("Generating test images...")
	testImages := generateTestImages(testDir)

	// Create hasher
	hasher := gopdq.NewPdqHasher()

	// Run benchmarks
	fmt.Println("\nRunning benchmarks...")
	results := make([]BenchmarkResult, 0, len(testImages))

	for _, imagePath := range testImages {
		result := benchmarkImage(hasher, imagePath)
		results = append(results, result)
		if verbose {
			fmt.Printf("✓ %s: %.2f hashes/sec\n", result.ImageName, result.HashesPerSec)
		}
	}

	// Display results
	displayResults(results)

	// Clean up test images
	fmt.Printf("\nCleaning up test images from %s...\n", testDir)
	if err := os.RemoveAll(testDir); err != nil {
		log.Printf("Warning: Failed to clean up test directory: %v", err)
	} else {
		fmt.Println("✓ Cleanup complete")
	}
}

func generateTestImages(dir string) []string {
	var paths []string

	// Different image sizes to test
	sizes := []struct {
		name   string
		width  int
		height int
	}{
		{"64x64", 64, 64},
		{"128x128", 128, 128},
		{"256x256", 256, 256},
		{"512x512", 512, 512},
		{"1024x1024", 1024, 1024},
		{"1920x1080", 1920, 1080},
		{"4096x4096", 4096, 4096},
	}

	// Different image types to test
	patterns := []struct {
		name string
		gen  func(width, height int) image.Image
	}{
		{"solid", generateSolidImage},
		{"gradient", generateGradientImage},
		{"checkerboard", generateCheckerboardImage},
		{"noise", generateNoiseImage},
		{"complex", generateComplexImage},
	}

	for _, size := range sizes {
		for _, pattern := range patterns {
			filename := fmt.Sprintf("%s_%s.png", size.name, pattern.name)
			path := filepath.Join(dir, filename)

			img := pattern.gen(size.width, size.height)
			if err := savePNG(img, path); err != nil {
				log.Printf("Failed to save %s: %v", filename, err)
				continue
			}
			paths = append(paths, path)
		}
	}

	return paths
}

func generateSolidImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	c := color.RGBA{128, 128, 128, 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func generateGradientImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			intensity := uint8(float64(x) / float64(width) * 255)
			img.Set(x, y, color.RGBA{intensity, intensity, intensity, 255})
		}
	}
	return img
}

func generateCheckerboardImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	squareSize := max(width/32, 8)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if (x/squareSize+y/squareSize)%2 == 0 {
				img.Set(x, y, color.RGBA{255, 255, 255, 255})
			} else {
				img.Set(x, y, color.RGBA{0, 0, 0, 255})
			}
		}
	}
	return img
}

func generateNoiseImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	rng := rand.New(rand.NewSource(42)) // Fixed seed for reproducibility

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			intensity := uint8(rng.Intn(256))
			img.Set(x, y, color.RGBA{intensity, intensity, intensity, 255})
		}
	}
	return img
}

func generateComplexImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Create a complex pattern with sinusoidal waves
			fx := float64(x) / float64(width) * 4 * math.Pi
			fy := float64(y) / float64(height) * 4 * math.Pi

			r := uint8((math.Sin(fx) + 1) / 2 * 255)
			g := uint8((math.Sin(fy) + 1) / 2 * 255)
			b := uint8((math.Sin(fx+fy) + 1) / 2 * 255)

			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
	return img
}

func savePNG(img image.Image, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}

func benchmarkImage(hasher *gopdq.PdqHasher, imagePath string) BenchmarkResult {
	// Number of runs for timing
	const numRuns = 10

	// Get image info first
	file, err := os.Open(imagePath)
	if err != nil {
		log.Printf("Failed to open %s: %v", imagePath, err)
		return BenchmarkResult{}
	}

	img, _, err := image.Decode(file)
	file.Close()
	if err != nil {
		log.Printf("Failed to decode %s: %v", imagePath, err)
		return BenchmarkResult{}
	}

	bounds := img.Bounds()
	pixels := bounds.Dx() * bounds.Dy()

	// Warm up
	hasher.FromFile(imagePath)

	// Benchmark runs
	var totalTime time.Duration
	var lastResult *gopdq.HashResult

	for i := 0; i < numRuns; i++ {
		start := time.Now()
		result, err := hasher.FromFile(imagePath)
		duration := time.Since(start)

		if err != nil {
			log.Printf("Failed to hash %s: %v", imagePath, err)
			return BenchmarkResult{}
		}

		totalTime += duration
		lastResult = result
	}

	avgTime := totalTime.Seconds() / float64(numRuns)
	hashesPerSec := 1.0 / avgTime

	return BenchmarkResult{
		ImageName:    filepath.Base(imagePath),
		ImageSize:    fmt.Sprintf("%dx%d", bounds.Dx(), bounds.Dy()),
		Pixels:       pixels,
		HashesPerSec: hashesPerSec,
		AvgHashTime:  avgTime,
		Quality:      lastResult.Quality,
		Hash:         lastResult.Hash.String(),
	}
}

func displayResults(results []BenchmarkResult) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("BENCHMARK RESULTS")
	fmt.Println(strings.Repeat("=", 80))

	fmt.Printf("%-25s %-12s %-10s %-12s %-8s %-8s\n",
		"Image", "Size", "Pixels", "Hashes/sec", "Avg(ms)", "Quality")
	fmt.Println(strings.Repeat("-", 80))

	totalHashes := 0.0
	totalPixels := 0

	for _, result := range results {
		fmt.Printf("%-25s %-12s %-10d %-12.2f %-8.1f %-8d\n",
			result.ImageName,
			result.ImageSize,
			result.Pixels,
			result.HashesPerSec,
			result.AvgHashTime*1000,
			result.Quality)

		totalHashes += result.HashesPerSec
		totalPixels += result.Pixels
	}

	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("Average hashes/sec: %.2f\n", totalHashes/float64(len(results)))
	fmt.Printf("Total pixels processed: %d\n", totalPixels)

	// Find fastest and slowest
	if len(results) > 0 {
		fastest := results[0]
		slowest := results[0]

		for _, result := range results[1:] {
			if result.HashesPerSec > fastest.HashesPerSec {
				fastest = result
			}
			if result.HashesPerSec < slowest.HashesPerSec {
				slowest = result
			}
		}

		fmt.Printf("\nFastest: %s (%.2f hashes/sec)\n", fastest.ImageName, fastest.HashesPerSec)
		fmt.Printf("Slowest: %s (%.2f hashes/sec)\n", slowest.ImageName, slowest.HashesPerSec)
	}

	fmt.Println("\nSample hashes:")
	for i, result := range results {
		if i < 5 { // Show first 5 hashes
			fmt.Printf("%s: %s\n", result.ImageName, result.Hash[:32]+"...")
		}
	}
}

func benchmarkDirectory(dirPath string, verbose bool, numHashes int) {
	// Check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		log.Fatalf("Directory does not exist: %s", dirPath)
	}

	// Find all image files
	imageFiles, err := findImageFiles(dirPath)
	if err != nil {
		log.Fatalf("Failed to scan directory: %v", err)
	}

	if len(imageFiles) == 0 {
		log.Fatalf("No image files found in directory: %s", dirPath)
	}

	fmt.Printf("Found %d image files in %s\n", len(imageFiles), dirPath)

	// Create hasher
	hasher := gopdq.NewPdqHasher()

	// Track statistics
	var totalReadSeconds, totalHashSeconds float64
	var numErrors, numSuccesses int
	var hashes []string

	// Determine how many images to process
	targetCount := numHashes
	if targetCount <= 0 || targetCount > len(imageFiles) {
		targetCount = len(imageFiles)
	}

	fmt.Printf("Processing %d images...\n", targetCount)
	startTime := time.Now()

	// Process images
	processedCount := 0
	for i := 0; processedCount < targetCount; i++ {
		// Loop through files if we need more than available
		fileIndex := i % len(imageFiles)
		imagePath := imageFiles[fileIndex]

		result, err := hasher.FromFile(imagePath)
		if err != nil {
			numErrors++
			if verbose {
				fmt.Printf("Error processing %s: %v\n", filepath.Base(imagePath), err)
			}
			continue
		}

		numSuccesses++
		processedCount++
		totalReadSeconds += float64(result.Stats.ReadSeconds)
		totalHashSeconds += float64(result.Stats.HashSeconds)
		hashes = append(hashes, result.Hash.String())

		if verbose {
			fmt.Printf("File: %s\n", filepath.Base(imagePath))
			fmt.Printf("Hash: %s\n", result.Hash.String())
			fmt.Printf("Quality: %d\n", result.Quality)
			fmt.Printf("Image pixels: %d\n", result.Stats.NumPixels)
			fmt.Printf("Read seconds: %.6f\n", result.Stats.ReadSeconds)
			fmt.Printf("Hash seconds: %.6f\n", result.Stats.HashSeconds)
			fmt.Println()
		} else if processedCount%100 == 0 || processedCount == targetCount {
			fmt.Printf("Processed %d/%d images\n", processedCount, targetCount)
		}

		// Break if we've processed all unique files and don't need repetition
		if numHashes <= 0 && fileIndex == len(imageFiles)-1 {
			break
		}
	}

	totalDuration := time.Since(startTime)

	// Display results in format similar to original C++ version
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("PHOTO COUNT:                    %d\n", numSuccesses)
	fmt.Printf("ERROR COUNT:                    %d\n", numErrors)
	fmt.Printf("TIME SPENT HASHING PHOTOS (SECONDS):   %.6f\n", totalHashSeconds)

	photosHashedPerSecond := 0.0
	if totalHashSeconds > 0 {
		photosHashedPerSecond = float64(numSuccesses) / totalHashSeconds
	}
	fmt.Printf("PHOTOS HASHED PER SECOND:       %.6f\n", photosHashedPerSecond)

	fmt.Printf("TIME SPENT READING PHOTOS (SECONDS):   %.6f\n", totalReadSeconds)

	photosReadPerSecond := 0.0
	if totalReadSeconds > 0 {
		photosReadPerSecond = float64(numSuccesses) / totalReadSeconds
	}
	fmt.Printf("PHOTOS READ PER SECOND:         %.6f\n", photosReadPerSecond)

	fmt.Printf("TOTAL BENCHMARK TIME (SECONDS): %.6f\n", totalDuration.Seconds())

	totalPhotosPerSecond := 0.0
	if totalDuration.Seconds() > 0 {
		totalPhotosPerSecond = float64(numSuccesses) / totalDuration.Seconds()
	}
	fmt.Printf("TOTAL PHOTOS PER SECOND:        %.6f\n", totalPhotosPerSecond)

	// Show sample hashes
	fmt.Println("\nSample hashes:")
	sampleCount := 5
	if len(hashes) < sampleCount {
		sampleCount = len(hashes)
	}
	for i := 0; i < sampleCount; i++ {
		fmt.Printf("%s: %s\n", filepath.Base(imageFiles[i]), hashes[i][:32]+"...")
	}
}

func findImageFiles(dirPath string) ([]string, error) {
	var imageFiles []string

	// Supported image extensions
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

