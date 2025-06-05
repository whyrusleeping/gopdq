package gopdq

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"time"
)

const (
	LUMA_FROM_R_COEFF         = 0.299
	LUMA_FROM_G_COEFF         = 0.587
	LUMA_FROM_B_COEFF         = 0.114
	DCT_MATRIX_SCALE_FACTOR   = math.Sqrt2 / 8.0 // sqrt(2.0 / 64.0)
	PDQ_NUM_JAROSZ_XY_PASSES  = 2
	PDQ_JAROSZ_WINDOW_SIZE_DIVISOR = 128
)

// HashResult contains the hash and quality metrics
type HashResult struct {
	Hash    *PdqHash256
	Quality int
	Stats   HashingStatistics
}

// HashingStatistics contains timing and performance metrics
type HashingStatistics struct {
	ReadSeconds  float64
	HashSeconds  float64
	NumPixels    int
	Source       string
}

// HashAndQuality is an internal struct for hash generation
type HashAndQuality struct {
	Hash    *PdqHash256
	Quality int
}

// PdqHasher is the main hasher implementation
type PdqHasher struct {
	dctMatrix []float64 // 16x64 matrix stored as 1D array
}

// NewPdqHasher creates a new PdqHasher instance
func NewPdqHasher() *PdqHasher {
	h := &PdqHasher{
		dctMatrix: make([]float64, 16*64),
	}
	h.computeDCTMatrix()
	return h
}

// computeDCTMatrix initializes the DCT transformation matrix
func (h *PdqHasher) computeDCTMatrix() {
	for i := 0; i < 16; i++ {
		for j := 0; j < 64; j++ {
			h.dctMatrix[i*64+j] = DCT_MATRIX_SCALE_FACTOR * math.Cos(math.Pi/2/64*float64(i+1)*float64(2*j+1))
		}
	}
}

// FromFile computes the PDQ hash from an image file
func (h *PdqHasher) FromFile(filePath string) (*HashResult, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	startTime := time.Now()
	
	// Decode image
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()
	width := min(bounds.Dx(), 1024)
	height := min(bounds.Dy(), 1024)

	// Resize if needed (simple nearest neighbor for now)
	var resized image.Image = img
	if bounds.Dx() > 1024 || bounds.Dy() > 1024 {
		resized = resizeImage(img, width, height)
	}

	readSeconds := time.Since(startTime).Seconds()
	numPixels := width * height

	// Process image
	hashStart := time.Now()
	
	buffer1 := make([]float64, height*width)
	buffer2 := make([]float64, height*width)
	buffer64x64 := make([]float64, 64*64)
	buffer16x16 := make([]float64, 16*16)

	h.fillFloatLumaFromImage(resized, buffer1)
	result := h.pdqHash256FromFloatLuma(buffer1, buffer2, height, width, buffer64x64, buffer16x16)

	hashSeconds := time.Since(hashStart).Seconds()

	return &HashResult{
		Hash:    result.Hash,
		Quality: result.Quality,
		Stats: HashingStatistics{
			ReadSeconds: readSeconds,
			HashSeconds: hashSeconds,
			NumPixels:   numPixels,
			Source:      filePath,
		},
	}, nil
}

// fillFloatLumaFromImage converts image pixels to luminance values
func (h *PdqHasher) fillFloatLumaFromImage(img image.Image, luma []float64) {
	bounds := img.Bounds()
	numCols := bounds.Dx()
	numRows := bounds.Dy()

	for row := 0; row < numRows; row++ {
		for col := 0; col < numCols; col++ {
			c := img.At(bounds.Min.X+col, bounds.Min.Y+row)
			r, g, b, _ := c.RGBA()
			// Convert to 8-bit values
			r8 := float64(r >> 8)
			g8 := float64(g >> 8)
			b8 := float64(b >> 8)
			
			luma[row*numCols+col] = LUMA_FROM_R_COEFF*r8 + LUMA_FROM_G_COEFF*g8 + LUMA_FROM_B_COEFF*b8
		}
	}
}

// pdqHash256FromFloatLuma generates the hash from luminance data
func (h *PdqHasher) pdqHash256FromFloatLuma(buffer1, buffer2 []float64, numRows, numCols int, buffer64x64, buffer16x16 []float64) HashAndQuality {
	windowSizeAlongRows := computeJaroszFilterWindowSize(numCols)
	windowSizeAlongCols := computeJaroszFilterWindowSize(numRows)

	jaroszFilterFloat(
		buffer1,
		buffer2,
		numRows,
		numCols,
		windowSizeAlongRows,
		windowSizeAlongCols,
		PDQ_NUM_JAROSZ_XY_PASSES,
	)

	decimateFloat(buffer1, numRows, numCols, buffer64x64)
	quality := computePDQImageDomainQualityMetric(buffer64x64)

	h.dct64To16(buffer64x64, buffer16x16)
	hash := pdqBuffer16x16ToBits(buffer16x16)

	return HashAndQuality{
		Hash:    hash,
		Quality: quality,
	}
}

// dct64To16 performs DCT transformation from 64x64 to 16x16
func (h *PdqHasher) dct64To16(A, B []float64) {
	// Temporary 16x64 matrix
	T := make([]float64, 16*64)

	// First multiplication: DCT * A
	for i := 0; i < 16; i++ {
		for j := 0; j < 64; j++ {
			var tij float64
			for k := 0; k < 64; k++ {
				tij += h.dctMatrix[i*64+k] * A[k*64+j]
			}
			T[i*64+j] = tij
		}
	}

	// Second multiplication: T * DCT^T
	for i := 0; i < 16; i++ {
		for j := 0; j < 16; j++ {
			var sumk float64
			for k := 0; k < 64; k++ {
				sumk += T[i*64+k] * h.dctMatrix[j*64+k]
			}
			B[i*16+j] = sumk
		}
	}
}

// pdqBuffer16x16ToBits converts DCT output to hash bits
func pdqBuffer16x16ToBits(dctOutput16x16 []float64) *PdqHash256 {
	hash := NewPdqHash256()
	
	// Calculate median using Torben's algorithm
	dctMedian := torbenMedian(dctOutput16x16)

	for i := 0; i < 16; i++ {
		for j := 0; j < 16; j++ {
			if dctOutput16x16[i*16+j] > dctMedian {
				hash.SetBit(i*16 + j)
			}
		}
	}
	return hash
}

// computePDQImageDomainQualityMetric calculates quality based on gradients
func computePDQImageDomainQualityMetric(buffer64x64 []float64) int {
	gradientSum := 0
	
	// Horizontal gradients
	for i := 0; i < 63; i++ {
		for j := 0; j < 64; j++ {
			u := buffer64x64[i*64+j]
			v := buffer64x64[(i+1)*64+j]
			d := int((u - v) * 100 / 255)
			if d < 0 {
				gradientSum -= d
			} else {
				gradientSum += d
			}
		}
	}
	
	// Vertical gradients
	for i := 0; i < 64; i++ {
		for j := 0; j < 63; j++ {
			u := buffer64x64[i*64+j]
			v := buffer64x64[i*64+j+1]
			d := int((u - v) * 100 / 255)
			if d < 0 {
				gradientSum -= d
			} else {
				gradientSum += d
			}
		}
	}
	
	quality := gradientSum / 90
	if quality > 100 {
		quality = 100
	}
	return quality
}

// decimateFloat downsamples from input resolution to 64x64
func decimateFloat(in []float64, inNumRows, inNumCols int, output []float64) {
	for i := 0; i < 64; i++ {
		ini := int((float64(i)+0.5)*float64(inNumRows)/64)
		for j := 0; j < 64; j++ {
			inj := int((float64(j)+0.5)*float64(inNumCols)/64)
			output[i*64+j] = in[ini*inNumCols+inj]
		}
	}
}

// jaroszFilterFloat applies Jarosz filter for image smoothing
func jaroszFilterFloat(buffer1, buffer2 []float64, numRows, numCols, windowSizeAlongRows, windowSizeAlongCols, nreps int) {
	for i := 0; i < nreps; i++ {
		boxAlongRowsFloat(buffer1, buffer2, numRows, numCols, windowSizeAlongRows)
		boxAlongColsFloat(buffer2, buffer1, numRows, numCols, windowSizeAlongCols)
	}
}

// boxAlongRowsFloat applies 1D box filter along rows
func boxAlongRowsFloat(input, output []float64, numRows, numCols, windowSize int) {
	for i := 0; i < numRows; i++ {
		box1DFloat(
			input,
			i*numCols,
			output,
			i*numCols,
			numCols,
			1,
			windowSize,
		)
	}
}

// boxAlongColsFloat applies 1D box filter along columns
func boxAlongColsFloat(input, output []float64, numRows, numCols, windowSize int) {
	for j := 0; j < numCols; j++ {
		box1DFloat(input, j, output, j, numRows, numCols, windowSize)
	}
}

// box1DFloat performs 1D box filtering
func box1DFloat(invec []float64, inStartOffset int, outVec []float64, outStartOffset int, vectorLength, stride, fullWindowSize int) {
	halfWindowSize := (fullWindowSize + 2) / 2
	phase1Nreps := halfWindowSize - 1
	phase2Nreps := fullWindowSize - halfWindowSize + 1
	phase3Nreps := vectorLength - fullWindowSize
	phase4Nreps := halfWindowSize - 1

	li := 0 // Index of left edge of read window
	ri := 0 // Index of right edge of read window
	oi := 0 // Index into output vector
	sum := 0.0
	currentWindowSize := 0

	// Phase 1: Initial accumulation
	for i := 0; i < phase1Nreps; i++ {
		sum += invec[inStartOffset+ri]
		currentWindowSize++
		ri += stride
	}

	// Phase 2: Initial writes with small window
	for i := 0; i < phase2Nreps; i++ {
		sum += invec[inStartOffset+ri]
		currentWindowSize++
		outVec[outStartOffset+oi] = sum / float64(currentWindowSize)
		ri += stride
		oi += stride
	}

	// Phase 3: Writes with full window
	for i := 0; i < phase3Nreps; i++ {
		sum += invec[inStartOffset+ri]
		sum -= invec[inStartOffset+li]
		outVec[outStartOffset+oi] = sum / float64(currentWindowSize)
		li += stride
		ri += stride
		oi += stride
	}

	// Phase 4: Final writes with small window
	for i := 0; i < phase4Nreps; i++ {
		sum -= invec[inStartOffset+li]
		currentWindowSize--
		outVec[outStartOffset+oi] = sum / float64(currentWindowSize)
		li += stride
		oi += stride
	}
}

// computeJaroszFilterWindowSize calculates the window size for Jarosz filter
func computeJaroszFilterWindowSize(dimensionSize int) int {
	return (dimensionSize + PDQ_JAROSZ_WINDOW_SIZE_DIVISOR - 1) / PDQ_JAROSZ_WINDOW_SIZE_DIVISOR
}

// torbenMedian implements Torben's algorithm for finding median
func torbenMedian(data []float64) float64 {
	// Create a copy to avoid modifying original
	arr := make([]float64, len(data))
	copy(arr, data)
	
	n := len(arr)
	if n == 0 {
		return 0
	}
	
	// For small arrays, use simple sorting
	if n < 30 {
		for i := 0; i < n-1; i++ {
			for j := i + 1; j < n; j++ {
				if arr[i] > arr[j] {
					arr[i], arr[j] = arr[j], arr[i]
				}
			}
		}
		if n%2 == 0 {
			return (arr[n/2-1] + arr[n/2]) / 2
		}
		return arr[n/2]
	}
	
	// Torben's algorithm for larger arrays
	low := 0
	high := n - 1
	median := (low + high) / 2
	
	for {
		if high <= low {
			return arr[median]
		}
		
		if high == low+1 {
			if arr[low] > arr[high] {
				arr[low], arr[high] = arr[high], arr[low]
			}
			return arr[median]
		}
		
		// Find median of low, middle, and high
		middle := (low + high) / 2
		if arr[middle] > arr[high] {
			arr[middle], arr[high] = arr[high], arr[middle]
		}
		if arr[low] > arr[high] {
			arr[low], arr[high] = arr[high], arr[low]
		}
		if arr[middle] > arr[low] {
			arr[middle], arr[low] = arr[low], arr[middle]
		}
		
		// Swap low item to middle position
		arr[middle], arr[low+1] = arr[low+1], arr[middle]
		
		// Nibble from each end towards middle
		ll := low + 1
		hh := high
		for {
			ll++
			for arr[low] > arr[ll] {
				ll++
			}
			hh--
			for arr[hh] > arr[low] {
				hh--
			}
			
			if hh < ll {
				break
			}
			
			arr[ll], arr[hh] = arr[hh], arr[ll]
		}
		
		// Swap middle item back
		arr[low], arr[hh] = arr[hh], arr[low]
		
		// Re-set active partition
		if hh <= median {
			low = ll
		}
		if hh >= median {
			high = hh - 1
		}
	}
}

// resizeImage performs simple nearest-neighbor image resizing
func resizeImage(src image.Image, width, height int) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	
	xRatio := float64(bounds.Dx()) / float64(width)
	yRatio := float64(bounds.Dy()) / float64(height)
	
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := int(float64(x) * xRatio)
			srcY := int(float64(y) * yRatio)
			dst.Set(x, y, src.At(bounds.Min.X+srcX, bounds.Min.Y+srcY))
		}
	}
	
	return dst
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}