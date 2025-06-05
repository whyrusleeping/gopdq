#ifndef SIMD_VECTORIZED_H
#define SIMD_VECTORIZED_H

#ifdef __cplusplus
extern "C" {
#endif

// SIMD-optimized vectorized division function
// Divides 8 float32 values element-wise: out[i] = num[i] / denom[i]
void simd_vectorized_div(float *out, const float *num, const float *denom);
void box1DFloatC(float *invec, float *outVec, int vectorLength, int stride,
                 int fullWindowSize);
void boxAlongColsFloatC(float *input, float *output, int numRows, int numCols,
                        int windowSize);
void jaroszFilterFloat(float *buffer1, float *buffer2, int numRows, int numCols,
                       int windowSizeAlongRows, int windowSizeAlongCols,
                       int nreps);

#ifdef __cplusplus
}
#endif

#endif // SIMD_VECTORIZED_H
