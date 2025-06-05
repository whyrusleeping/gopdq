#include "simd_vectorized.h"
#include <immintrin.h>  // For AVX2 intrinsics


// Fallback implementation for systems without AVX2
/*
#ifndef __AVX2__
void simd_vectorized_div(float* out, const float* num, const float* denom) {
        out[0] = num[0] / denom[0];
        out[1] = num[1] / denom[1];
        out[2] = num[2] / denom[2];
        out[3] = num[3] / denom[3];
        out[4] = num[4] / denom[4];
        out[5] = num[5] / denom[5];
        out[6] = num[6] / denom[6];
        out[7] = num[7] / denom[7];
}
#else
*/
void simd_vectorized_div(float* out, const float* num, const float* denom) {
    // Load 8 float values from num and denom arrays into AVX2 256-bit registers
    __m256 num_vec = _mm256_loadu_ps(num);
    __m256 denom_vec = _mm256_loadu_ps(denom);
    
    // Perform element-wise division using AVX2
    __m256 result_vec = _mm256_div_ps(num_vec, denom_vec);
    
    // Store the result back to the output array
    _mm256_storeu_ps(out, result_vec);
}

void simd_vectorized_mul(float* out, const float* num, const float* op) {
    // Load 8 float values from num and denom arrays into AVX2 256-bit registers
    __m256 num_vec = _mm256_loadu_ps(num);
    __m256 op_vec = _mm256_loadu_ps(op);
    
    __m256 result_vec = _mm256_mul_ps(num_vec, op_vec);
    
    // Store the result back to the output array
    _mm256_storeu_ps(out, result_vec);
}
//#endif

void box1DFloatC(float* invec, float * outVec,  int vectorLength, int  stride, int fullWindowSize ) {
	int halfWindowSize = (fullWindowSize + 2) / 2;
	int phase1Nreps = halfWindowSize - 1;
	int phase2Nreps = fullWindowSize - halfWindowSize + 1;
	int phase3Nreps = vectorLength - fullWindowSize;
	int phase4Nreps = halfWindowSize - 1;

	int li = 0; // Index of left edge of read window
	int ri = 0; // Index of right edge of read window
	int oi = 0;// Index into output vector
	float sum = 0.0;
	float currentWindowSize = 0;

	// Phase 1: Initial accumulation
	for (int i = 0; i < phase1Nreps; i++) {
		sum += invec[ri];
		currentWindowSize += 1;
		ri += stride;
	}

	// Phase 2: Initial writes with small window
	for (int  i = 0; i < phase2Nreps; i++ ) {
		sum += invec[ri];
		currentWindowSize++;
		outVec[oi] = sum / currentWindowSize;
		ri += stride;
		oi += stride;
	}

	// Phase 3: Writes with full window
	int i = 0;
	/*
	float buf[8] = {0};
	float out[8] = {0};
	float denom[8];
	for (int j = 0; j < 8; j++) {
		denom[j] = currentWindowSize;
	}

	for (; i+8 < phase3Nreps; i += 8) {
		//sum += invec[ri]
		//sum -= invec[li]

		buf[0] = sum + invec[ri+(stride*0)] - invec[li+(stride*0)];
		buf[1] = buf[0] + invec[ri+(stride*1)] - invec[li+(stride*1)];
		buf[2] = buf[1] + invec[ri+(stride*2)] - invec[li+(stride*2)];
		buf[3] = buf[2] + invec[ri+(stride*3)] - invec[li+(stride*3)];
		buf[4] = buf[3] + invec[ri+(stride*4)] - invec[li+(stride*4)];
		buf[5] = buf[4] + invec[ri+(stride*5)] - invec[li+(stride*5)];
		buf[6] = buf[5] + invec[ri+(stride*6)] - invec[li+(stride*6)];
		buf[7] = buf[6] + invec[ri+(stride*7)] - invec[li+(stride*7)];

		simd_vectorized_div(out, buf, denom);
		outVec[oi] = out[0];
		outVec[oi+(stride*1)] = out[1];
		outVec[oi+(stride*2)] = out[2];
		outVec[oi+(stride*3)] = out[3];
		outVec[oi+(stride*4)] = out[4];
		outVec[oi+(stride*5)] = out[5];
		outVec[oi+(stride*6)] = out[6];
		outVec[oi+(stride*7)] = out[7];

		li += stride * 8;
		ri += stride * 8;
		oi += stride * 8;

		sum = buf[7];
	}
	*/

	float buf[8];
	float denom = 1 / currentWindowSize;
	/*
	float denom_buf[8];
	for (int j = 0; j < 8; j++) {
		denom_buf[j] = denom;
	}
	float out[8];
	*/

	for (; i < phase3Nreps-8; i += 8) {
		buf[0] = sum + invec[ri+(stride*0)] - invec[li+(stride*0)];
		buf[1] = buf[0] + invec[ri+(stride*1)] - invec[li+(stride*1)];
		buf[2] = buf[1] + invec[ri+(stride*2)] - invec[li+(stride*2)];
		buf[3] = buf[2] + invec[ri+(stride*3)] - invec[li+(stride*3)];
		buf[4] = buf[3] + invec[ri+(stride*4)] - invec[li+(stride*4)];
		buf[5] = buf[4] + invec[ri+(stride*5)] - invec[li+(stride*5)];
		buf[6] = buf[5] + invec[ri+(stride*6)] - invec[li+(stride*6)];
		buf[7] = buf[6] + invec[ri+(stride*7)] - invec[li+(stride*7)];

		outVec[oi] = buf[0] * denom;
		outVec[oi+(stride*1)] = buf[1] * denom;
		outVec[oi+(stride*2)] = buf[2] * denom;
		outVec[oi+(stride*3)] = buf[3] * denom;
		outVec[oi+(stride*4)] = buf[4] * denom;
		outVec[oi+(stride*5)] = buf[5] * denom;
		outVec[oi+(stride*6)] = buf[6] * denom;
		outVec[oi+(stride*7)] = buf[7] * denom;

		/*
		simd_vectorized_mul(out, buf, denom_buf);
		outVec[oi] = out[0];
		outVec[oi+(stride*1)] = out[1];
		outVec[oi+(stride*2)] = out[2];
		outVec[oi+(stride*3)] = out[3];
		outVec[oi+(stride*4)] = out[4];
		outVec[oi+(stride*5)] = out[5];
		outVec[oi+(stride*6)] = out[6];
		outVec[oi+(stride*7)] = out[7];
		*/

		li += stride * 8;
		ri += stride * 8;
		oi += stride * 8;

		sum = buf[7];
	}

	for (; i < phase3Nreps; i++) {
		sum += invec[ri];
		sum -= invec[li];
		outVec[oi] = sum / currentWindowSize;
		li += stride;
		ri += stride;
		oi += stride;
	}

	// Phase 4: Final writes with small window
	for (int i = 0; i < phase4Nreps; i++) {
		sum -= invec[li];
		currentWindowSize--;
		outVec[oi] = sum / currentWindowSize;
		li += stride;
		oi += stride;
	}
}

void boxAlongColsFloatC(float *input, float *output ,int numRows, int numCols, int windowSize ) {
	for (int j = 0; j < numCols; j++) {
		box1DFloatC(&input[j], &output[j], numRows, numCols, windowSize);
	}
}

// boxAlongRowsFloat applies 1D box filter along rows
void boxAlongRowsFloatC(float *input, float *output, int numRows, int numCols, int windowSize) {
	for (int i = 0; i < numRows; i++) {
		box1DFloatC(
			&input[i*numCols],
			&output[i*numCols],
			numCols,
			1,
			windowSize
		);
	}
}

// jaroszFilterFloat applies Jarosz filter for image smoothing
void jaroszFilterFloat(float *buffer1, float* buffer2 , int numRows, int numCols, int windowSizeAlongRows, int windowSizeAlongCols, int nreps) {
	for ( int i = 0; i < nreps; i++) {
		boxAlongRowsFloatC(buffer1, buffer2, numRows, numCols, windowSizeAlongRows);
		boxAlongColsFloatC(buffer2, buffer1, numRows, numCols, windowSizeAlongCols);
	}
}

