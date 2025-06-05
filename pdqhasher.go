package gopdq

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"os"

	ljpeg "github.com/pixiv/go-libjpeg/jpeg"
)

const (
	LUMA_FROM_R_COEFF              = 0.299
	LUMA_FROM_G_COEFF              = 0.587
	LUMA_FROM_B_COEFF              = 0.114
	PDQ_NUM_JAROSZ_XY_PASSES       = 2
	PDQ_JAROSZ_WINDOW_SIZE_DIVISOR = 128
)

// HashResult contains the hash and quality metrics
type HashResult struct {
	Hash    *PdqHash256
	Quality int
}

// HashAndQuality is an internal struct for hash generation
type HashAndQuality struct {
	Hash    *PdqHash256
	Quality int
}

// PdqHasher is the main hasher implementation
type PdqHasher struct {
	dctMatrix []float32 // 16x64 matrix stored as 1D array
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
	matrixScaleFactor := float32(math.Sqrt(2.0 / 64.0))
	for i := 0; i < 16; i++ {
		for j := 0; j < 64; j++ {
			h.dctMatrix[i*64+j] = matrixScaleFactor * float32(math.Cos((math.Pi/2.0/64.0)*float64(i+1)*float64(2*j+1)))
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

	return h.FromReader(file)
}

func DecodeJpeg(r io.Reader) (image.Image, error) {
	var img image.Image
	if ljpeg.SupportRGBA() {
		ljimg, err := ljpeg.DecodeIntoRGBA(r, &ljpeg.DecoderOptions{
			DCTMethod:              ljpeg.DCTIFast,
			DisableFancyUpsampling: false,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to decode image: %w", err)
		}
		img = ljimg
	} else {
		ljimg, err := ljpeg.Decode(r, &ljpeg.DecoderOptions{
			DCTMethod:              ljpeg.DCTIFast,
			DisableFancyUpsampling: false,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to decode image: %w", err)
		}
		img = ljimg
	}

	return img, nil
}

func (h *PdqHasher) FromReader(r io.Reader) (*HashResult, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}

	return h.HashImage(img)
}

func (h *PdqHasher) HashImage(img image.Image) (*HashResult, error) {
	//width := min(bounds.Dx(), 1024)
	//height := min(bounds.Dy(), 1024)

	var resized image.Image = img
	// Resize if needed (simple nearest neighbor for now)
	/*
		bounds := img.Bounds()
		if bounds.Dx() > 1024 || bounds.Dy() > 1024 {
			resized = resize.Resize(uint(width), uint(height), img, resize.NearestNeighbor)
			//resized = resizeImage(img, width, height)
		}
	*/

	width := resized.Bounds().Dx()
	height := resized.Bounds().Dy()

	// Process image

	buffer1 := make([]float32, height*width)
	buffer2 := make([]float32, height*width)
	buffer64x64 := make([]float32, 64*64)
	buffer16x16 := make([]float32, 16*16)

	h.fillFloatLumaFromImage(resized, buffer1)
	result := h.pdqHash256FromFloatLuma(buffer1, buffer2, height, width, buffer64x64, buffer16x16)

	return &HashResult{
		Hash:    result.Hash,
		Quality: result.Quality,
	}, nil
}

// fillFloatLumaFromImage converts image pixels to luminance values
func (h *PdqHasher) fillFloatLumaFromImage(img image.Image, luma []float32) {
	bounds := img.Bounds()
	numCols := bounds.Dx()
	numRows := bounds.Dy()

	var rgbaImg *image.RGBA
	if rgbaSrc, ok := img.(*image.RGBA); ok {
		rgbaImg = rgbaSrc
	} else {
		rgbaImg = image.NewRGBA(img.Bounds())
		draw.Draw(rgbaImg, rgbaImg.Bounds(), img, img.Bounds().Min, draw.Src)
	}

	// Now access raw RGBA data
	stride := rgbaImg.Stride

	for row := 0; row < numRows; row++ {
		for col := 0; col < numCols; col++ {
			offs := (row * stride) + (col * 4)
			r8 := float32(rgbaImg.Pix[offs])
			g8 := float32(rgbaImg.Pix[offs+1])
			b8 := float32(rgbaImg.Pix[offs+2])

			luma[row*numCols+col] = LUMA_FROM_R_COEFF*r8 + LUMA_FROM_G_COEFF*g8 + LUMA_FROM_B_COEFF*b8
		}
	}
}

// pdqHash256FromFloatLuma generates the hash from luminance data
func (h *PdqHasher) pdqHash256FromFloatLuma(buffer1, buffer2 []float32, numRows, numCols int, buffer64x64, buffer16x16 []float32) HashAndQuality {
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
func (h *PdqHasher) dct64To16(A, B []float32) {
	// Temporary 16x64 matrix
	T := make([]float32, 16*64)

	// First multiplication: DCT * A
	for i := 0; i < 16; i++ {
		for j := 0; j < 64; j++ {
			var tij float32
			for k := 0; k < 64; k++ {
				tij += h.dctMatrix[i*64+k] * A[k*64+j]
			}
			T[i*64+j] = tij
		}
	}

	// Second multiplication: T * DCT^T
	for i := 0; i < 16; i++ {
		for j := 0; j < 16; j++ {
			var sumk float32
			for k := 0; k < 64; k++ {
				sumk += T[i*64+k] * h.dctMatrix[j*64+k]
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
		ini := int((float32(i) + 0.5) * float32(inNumRows) / 64)
		for j := 0; j < 64; j++ {
			inj := int((float32(j) + 0.5) * float32(inNumCols) / 64)
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
			input[i*numCols:],
			output[i*numCols:],
			numCols,
			1,
			windowSize,
		)
	}
}

// boxAlongColsFloat applies 1D box filter along columns
func boxAlongColsFloat(input, output []float32, numRows, numCols, windowSize int) {
	for j := 0; j < numCols; j++ {
		box1DFloat(input[j:], output[j:], numRows, numCols, windowSize)
	}
}

// box1DFloat performs 1D box filtering
func box1DFloat(invec []float32, outVec []float32, vectorLength, stride, fullWindowSize int) {
	halfWindowSize := (fullWindowSize + 2) / 2
	phase1Nreps := halfWindowSize - 1
	phase2Nreps := fullWindowSize - halfWindowSize + 1
	phase3Nreps := vectorLength - fullWindowSize
	phase4Nreps := halfWindowSize - 1

	li := 0 // Index of left edge of read window
	ri := 0 // Index of right edge of read window
	oi := 0 // Index into output vector
	sum := float32(0.0)
	currentWindowSize := float32(0)

	// Phase 1: Initial accumulation
	for i := 0; i < phase1Nreps; i++ {
		sum += invec[ri]
		currentWindowSize++
		ri += stride
	}

	// Phase 2: Initial writes with small window
	for i := 0; i < phase2Nreps; i++ {
		sum += invec[ri]
		currentWindowSize++
		outVec[oi] = sum / currentWindowSize
		ri += stride
		oi += stride
	}

	// Phase 3: Writes with full window
	var i int
	/*
		buf := make([]float32, 8)
		denom := make([]float32, 8)
		out := make([]float32, 8)
		for j := 0; j < 8; j++ {
			denom[j] = currentWindowSize
		}

		for ; i+8 < phase3Nreps; i += 8 {
			//sum += invec[ri]
			//sum -= invec[li]

			buf[0] = sum + invec[ri+(stride*0)] - invec[li+(stride*0)]
			buf[1] = buf[0] + invec[ri+(stride*1)] - invec[li+(stride*1)]
			buf[2] = buf[1] + invec[ri+(stride*2)] - invec[li+(stride*2)]
			buf[3] = buf[2] + invec[ri+(stride*3)] - invec[li+(stride*3)]
			buf[4] = buf[3] + invec[ri+(stride*4)] - invec[li+(stride*4)]
			buf[5] = buf[4] + invec[ri+(stride*5)] - invec[li+(stride*5)]
			buf[6] = buf[5] + invec[ri+(stride*6)] - invec[li+(stride*6)]
			buf[7] = buf[6] + invec[ri+(stride*7)] - invec[li+(stride*7)]

			/*
				outVec[oi] = buf[0] / currentWindowSize
				outVec[oi+(stride*1)] = buf[1] / currentWindowSize
				outVec[oi+(stride*2)] = buf[2] / currentWindowSize
				outVec[oi+(stride*3)] = buf[3] / currentWindowSize
				outVec[oi+(stride*4)] = buf[4] / currentWindowSize
				outVec[oi+(stride*5)] = buf[5] / currentWindowSize
				outVec[oi+(stride*6)] = buf[6] / currentWindowSize
				outVec[oi+(stride*7)] = buf[7] / currentWindowSize
	*/ /*
			vectorizedDiv(out, buf, denom)
			outVec[oi] = out[0]
			outVec[oi+(stride*1)] = out[1]
			outVec[oi+(stride*2)] = out[2]
			outVec[oi+(stride*3)] = out[3]
			outVec[oi+(stride*4)] = out[4]
			outVec[oi+(stride*5)] = out[5]
			outVec[oi+(stride*6)] = out[6]
			outVec[oi+(stride*7)] = out[7]

			li += stride * 8
			ri += stride * 8
			oi += stride * 8

			sum = buf[7]
		}
	*/

	denom := 1 / currentWindowSize
	for ; i < phase3Nreps; i++ {
		sum += invec[ri]
		sum -= invec[li]
		outVec[oi] = sum * denom
		li += stride
		ri += stride
		oi += stride
	}

	// Phase 4: Final writes with small window
	for i := 0; i < phase4Nreps; i++ {
		sum -= invec[li]
		currentWindowSize--
		outVec[oi] = sum / currentWindowSize
		li += stride
		oi += stride
	}
}

// computeJaroszFilterWindowSize calculates the window size for Jarosz filter
func computeJaroszFilterWindowSize(dimensionSize int) int {
	return (dimensionSize + PDQ_JAROSZ_WINDOW_SIZE_DIVISOR - 1) / PDQ_JAROSZ_WINDOW_SIZE_DIVISOR
}

// torbenMedian implements Torben's median algorithm
// This is a direct port of the C++ implementation
func torbenMedian(m []float32) float32 {
	n := len(m)
	if n == 0 {
		return 0
	}

	// Create a copy to avoid modifying original
	arr := make([]float32, n)
	copy(arr, m)

	min := arr[0]
	max := arr[0]
	for i := 1; i < n; i++ {
		if arr[i] < min {
			min = arr[i]
		}
		if arr[i] > max {
			max = arr[i]
		}
	}

	var less, greater, equal int
	var maxltguess, mingtguess, guess float32

	for {
		guess = (min + max) / 2
		less = 0
		greater = 0
		equal = 0
		maxltguess = min
		mingtguess = max

		for i := 0; i < n; i++ {
			if arr[i] < guess {
				less++
				if arr[i] > maxltguess {
					maxltguess = arr[i]
				}
			} else if arr[i] > guess {
				greater++
				if arr[i] < mingtguess {
					mingtguess = arr[i]
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

	// Calculate the final result based on the C++ logic
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
