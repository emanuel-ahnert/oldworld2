// Package procgen generates the world deterministically from a seed.
// Everything here is pure: the same seed and coordinates always give the same result,
// independent of the order in which chunks are generated.
package procgen

import "math"

// hash mixes a seed, a salt and two coordinates into a well-distributed 64-bit value
// (splitmix64 finalizer).
func hash(seed int64, salt uint64, x, y int) uint64 {
	h := uint64(seed) ^ salt*0x9E3779B97F4A7C15
	h ^= uint64(uint32(x)) * 0xBF58476D1CE4E5B9
	h ^= uint64(uint32(y)) * 0x94D049BB133111EB
	h ^= h >> 30
	h *= 0xBF58476D1CE4E5B9
	h ^= h >> 27
	h *= 0x94D049BB133111EB
	h ^= h >> 31
	return h
}

// rand01 returns a deterministic value in [0, 1) for a tile.
func rand01(seed int64, salt uint64, x, y int) float64 {
	return float64(hash(seed, salt, x, y)>>11) / (1 << 53)
}

func smooth(t float64) float64 { return t * t * (3 - 2*t) }

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

// valueNoise is smooth 2D value noise in [0, 1) with lattice spacing 1.
func valueNoise(seed int64, salt uint64, x, y float64) float64 {
	x0, y0 := math.Floor(x), math.Floor(y)
	ix, iy := int(x0), int(y0)
	tx, ty := smooth(x-x0), smooth(y-y0)
	a := rand01(seed, salt, ix, iy)
	b := rand01(seed, salt, ix+1, iy)
	c := rand01(seed, salt, ix, iy+1)
	d := rand01(seed, salt, ix+1, iy+1)
	return lerp(lerp(a, b, tx), lerp(c, d, tx), ty)
}

// fbm layers several octaves of value noise. scale is the size of the largest features in tiles.
// The result is roughly in [0, 1), centered around 0.5.
func fbm(seed int64, salt uint64, x, y int, scale float64, octaves int) float64 {
	sum, amp, norm := 0.0, 1.0, 0.0
	fx, fy := float64(x)/scale, float64(y)/scale
	for o := 0; o < octaves; o++ {
		sum += amp * valueNoise(seed, salt+uint64(o)*7919, fx, fy)
		norm += amp
		amp *= 0.5
		fx *= 2
		fy *= 2
	}
	return sum / norm
}
