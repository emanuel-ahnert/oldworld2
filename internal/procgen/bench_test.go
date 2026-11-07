package procgen

import "testing"

func BenchmarkGenerateChunk(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateChunk(1, i%40-20, i/40%40-20)
	}
}

// Walking through the world: chunks share one terrain cache, like in the game.
func BenchmarkGenerateChunkShared(b *testing.B) {
	t := NewTerrainCache(1)
	for i := 0; i < b.N; i++ {
		t.Generate(i%40-20, i/40%40-20)
	}
}

func BenchmarkGenerateTerrain(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generateTerrain(1, i%40-20, i/40%40-20)
	}
}
