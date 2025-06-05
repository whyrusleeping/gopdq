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
	LUMA_FROM_R_COEFF         = float32(0.299)
	LUMA_FROM_G_COEFF         = float32(0.587)
	LUMA_FROM_B_COEFF         = float32(0.114)
	PDQ_NUM_JAROSZ_XY_PASSES  = 2
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
	dctMatrix []float32 // 16x64 matrix stored as 1D array, using float32 like C++
}

// NewPdqHasher creates a new PdqHasher instance
func NewPdqHasher() *PdqHasher {
	h := &PdqHasher{
		dctMatrix: make([]float32, 16*64),
	}
	h.computeDCTMatrix()
	return h
}

// computeDCTMatrix initializes the DCT transformation matrix
func (h *PdqHasher) computeDCTMatrix() {
	const numRows = 16
	const numCols = 64
	matrixScaleFactor := float32(math.Sqrt(2.0 / float64(numCols)))
	
	for i := 0; i < numRows; i++ {
		for j := 0; j < numCols; j++ {
			h.dctMatrix[i*numCols+j] = matrixScaleFactor * float32(math.Cos((math.Pi/2.0/float64(numCols))*float64(i+1)*float64(2*j+1)))
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
	width := bounds.Dx()
	height := bounds.Dy()

	// Use original image without automatic resizing to match C++ behavior
	var resized image.Image = img

	readSeconds := time.Since(startTime).Seconds()
	numPixels := width * height

	// Process image
	hashStart := time.Now()
	
	buffer1 := make([]float32, height*width)
	buffer2 := make([]float32, height*width)
	buffer64x64 := make([]float32, 64*64)
	buffer16x16 := make([]float32, 16*16)

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
func (h *PdqHasher) fillFloatLumaFromImage(img image.Image, luma []float32) {
	bounds := img.Bounds()
	numCols := bounds.Dx()
	numRows := bounds.Dy()

	for row := 0; row < numRows; row++ {
		for col := 0; col < numCols; col++ {
			c := img.At(bounds.Min.X+col, bounds.Min.Y+row)
			r, g, b, _ := c.RGBA()
			// Convert exactly like C++ uint8_t values (truncate, don't round)
			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8) 
			b8 := uint8(b >> 8)
			
			luma[row*numCols+col] = LUMA_FROM_R_COEFF*float32(r8) + LUMA_FROM_G_COEFF*float32(g8) + LUMA_FROM_B_COEFF*float32(b8)
		}
	}
}

// pdqHash256FromFloatLuma generates the hash from luminance data
func (h *PdqHasher) pdqHash256FromFloatLuma(buffer1, buffer2 []float32, numRows, numCols int, buffer64x64, buffer16x16 []float32) HashAndQuality {
	windowSizeAlongRows := computeJaroszFilterWindowSize(numCols, 64)
	windowSizeAlongCols := computeJaroszFilterWindowSize(numRows, 64)

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

// dct64To16 performs DCT transformation from 64x64 to 16x16 using exact C++ algorithm
func (h *PdqHasher) dct64To16(A, B []float32) {
	// Temporary 16x64 matrix
	T := make([]float32, 16*64)

	// First multiplication: T = D * A (exactly like C++)
	for i := 0; i < 16; i++ {
		for j := 0; j < 64; j++ {
			var sumk float32 = 0.0
			
			// Unrolled loop like C++ version
			for k := 0; k < 64; k += 16 {
				sumk += h.dctMatrix[i*64+k+0] * A[(k+0)*64+j]
				sumk += h.dctMatrix[i*64+k+1] * A[(k+1)*64+j]
				sumk += h.dctMatrix[i*64+k+2] * A[(k+2)*64+j]
				sumk += h.dctMatrix[i*64+k+3] * A[(k+3)*64+j]
				sumk += h.dctMatrix[i*64+k+4] * A[(k+4)*64+j]
				sumk += h.dctMatrix[i*64+k+5] * A[(k+5)*64+j]
				sumk += h.dctMatrix[i*64+k+6] * A[(k+6)*64+j]
				sumk += h.dctMatrix[i*64+k+7] * A[(k+7)*64+j]
				sumk += h.dctMatrix[i*64+k+8] * A[(k+8)*64+j]
				sumk += h.dctMatrix[i*64+k+9] * A[(k+9)*64+j]
				sumk += h.dctMatrix[i*64+k+10] * A[(k+10)*64+j]
				sumk += h.dctMatrix[i*64+k+11] * A[(k+11)*64+j]
				sumk += h.dctMatrix[i*64+k+12] * A[(k+12)*64+j]
				sumk += h.dctMatrix[i*64+k+13] * A[(k+13)*64+j]
				sumk += h.dctMatrix[i*64+k+14] * A[(k+14)*64+j]
				sumk += h.dctMatrix[i*64+k+15] * A[(k+15)*64+j]
			}
			T[i*64+j] = sumk
		}
	}

	// Second multiplication: B = T * D^T (exactly like C++)
	for i := 0; i < 16; i++ {
		for j := 0; j < 16; j++ {
			var sumk float32 = 0.0
			
			// Unrolled loop like C++ version
			for k := 0; k < 64; k += 16 {
				sumk += T[i*64+k+0] * h.dctMatrix[j*64+k+0]
				sumk += T[i*64+k+1] * h.dctMatrix[j*64+k+1]
				sumk += T[i*64+k+2] * h.dctMatrix[j*64+k+2]
				sumk += T[i*64+k+3] * h.dctMatrix[j*64+k+3]
				sumk += T[i*64+k+4] * h.dctMatrix[j*64+k+4]
				sumk += T[i*64+k+5] * h.dctMatrix[j*64+k+5]
				sumk += T[i*64+k+6] * h.dctMatrix[j*64+k+6]
				sumk += T[i*64+k+7] * h.dctMatrix[j*64+k+7]
				sumk += T[i*64+k+8] * h.dctMatrix[j*64+k+8]
				sumk += T[i*64+k+9] * h.dctMatrix[j*64+k+9]
				sumk += T[i*64+k+10] * h.dctMatrix[j*64+k+10]
				sumk += T[i*64+k+11] * h.dctMatrix[j*64+k+11]
				sumk += T[i*64+k+12] * h.dctMatrix[j*64+k+12]
				sumk += T[i*64+k+13] * h.dctMatrix[j*64+k+13]
				sumk += T[i*64+k+14] * h.dctMatrix[j*64+k+14]
				sumk += T[i*64+k+15] * h.dctMatrix[j*64+k+15]
			}
			B[i*16+j] = sumk
		}
	}
}

// pdqBuffer16x16ToBits converts DCT output to hash bits
func pdqBuffer16x16ToBits(dctOutput16x16 []float32) *PdqHash256 {
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
func computePDQImageDomainQualityMetric(buffer64x64 []float32) int {
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
func decimateFloat(in []float32, inNumRows, inNumCols int, output []float32) {
	for i := 0; i < 64; i++ {
		ini := int((float32(i)+0.5)*float32(inNumRows)/64)
		for j := 0; j < 64; j++ {
			inj := int((float32(j)+0.5)*float32(inNumCols)/64)
			output[i*64+j] = in[ini*inNumCols+inj]
		}
	}
}

// jaroszFilterFloat applies Jarosz filter for image smoothing
func jaroszFilterFloat(buffer1, buffer2 []float32, numRows, numCols, windowSizeAlongRows, windowSizeAlongCols, nreps int) {
	for i := 0; i < nreps; i++ {
		boxAlongRowsFloat(buffer1, buffer2, numRows, numCols, windowSizeAlongRows)
		boxAlongColsFloat(buffer2, buffer1, numRows, numCols, windowSizeAlongCols)
	}
}

// boxAlongRowsFloat applies 1D box filter along rows
func boxAlongRowsFloat(input, output []float32, numRows, numCols, windowSize int) {
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
func boxAlongColsFloat(input, output []float32, numRows, numCols, windowSize int) {
	for j := 0; j < numCols; j++ {
		box1DFloat(input, j, output, j, numRows, numCols, windowSize)
	}
}

// box1DFloat performs 1D box filtering
func box1DFloat(invec []float32, inStartOffset int, outVec []float32, outStartOffset int, vectorLength, stride, fullWindowSize int) {
	halfWindowSize := (fullWindowSize + 2) / 2
	phase1Nreps := halfWindowSize - 1
	phase2Nreps := fullWindowSize - halfWindowSize + 1
	phase3Nreps := vectorLength - fullWindowSize
	phase4Nreps := halfWindowSize - 1

	li := 0 // Index of left edge of read window
	ri := 0 // Index of right edge of read window
	oi := 0 // Index into output vector
	var sum float32 = 0.0
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
		outVec[outStartOffset+oi] = sum / float32(currentWindowSize)
		ri += stride
		oi += stride
	}

	// Phase 3: Writes with full window
	for i := 0; i < phase3Nreps; i++ {
		sum += invec[inStartOffset+ri]
		sum -= invec[inStartOffset+li]
		outVec[outStartOffset+oi] = sum / float32(currentWindowSize)
		li += stride
		ri += stride
		oi += stride
	}

	// Phase 4: Final writes with small window
	for i := 0; i < phase4Nreps; i++ {
		sum -= invec[inStartOffset+li]
		currentWindowSize--
		outVec[outStartOffset+oi] = sum / float32(currentWindowSize)
		li += stride
		oi += stride
	}
}

// computeJaroszFilterWindowSize calculates the window size for Jarosz filter
func computeJaroszFilterWindowSize(oldDimension, newDimension int) int {
	return (oldDimension + 2*newDimension - 1) / (2 * newDimension)
}

// torbenMedian implements the exact Torben's algorithm from Facebook's C++ implementation
func torbenMedian(data []float32) float32 {
	n := len(data)
	if n == 0 {
		return 0
	}
	
	// Find min and max
	min := data[0]
	max := data[0]
	for i := 1; i < n; i++ {
		if data[i] < min {
			min = data[i]
		}
		if data[i] > max {
			max = data[i]
		}
	}
	
	for {
		guess := (min + max) / 2.0
		less := 0
		greater := 0
		equal := 0
		maxltguess := min
		mingtguess := max
		
		for i := 0; i < n; i++ {
			if data[i] < guess {
				less++
				if data[i] > maxltguess {
					maxltguess = data[i]
				}
			} else if data[i] > guess {
				greater++
				if data[i] < mingtguess {
					mingtguess = data[i]
				}
			} else {
				equal++
			}
		}
		
		if less <= (n+1)/2 && greater <= (n+1)/2 {
			break
		} else if less > greater {
			max = maxltguess
		} else {
			min = mingtguess
		}
	}
	
	// Final determination - exact C++ logic
	guess := (min + max) / 2.0
	less := 0
	greater := 0
	equal := 0
	maxltguess := min
	mingtguess := max
	
	for i := 0; i < n; i++ {
		if data[i] < guess {
			less++
			if data[i] > maxltguess {
				maxltguess = data[i]
			}
		} else if data[i] > guess {
			greater++
			if data[i] < mingtguess {
				mingtguess = data[i]
			}
		} else {
			equal++
		}
	}
	
	if less >= (n+1)/2 {
		return maxltguess
	} else if less+equal >= (n+1)/2 {
		return guess
	} else {
		return mingtguess
	}
}

// resizeImage performs simple nearest-neighbor image resizing
func resizeImage(src image.Image, width, height int) image.Image {
	bounds := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	
	xRatio := float32(bounds.Dx()) / float32(width)
	yRatio := float32(bounds.Dy()) / float32(height)
	
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := int(float32(x) * xRatio)
			srcY := int(float32(y) * yRatio)
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