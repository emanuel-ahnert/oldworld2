package r3d

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// meshData keeps the vertex arrays of generated meshes alive: raylib's Mesh points at them.
var meshData [][]float32

// Prism makes a flat-shaded mesh by extruding a polygon in the XY plane (counter-clockwise)
// along Z, centered, depth long. caps splits the polygon into convex pieces for the end caps
// (as lists of point indices); nil means the polygon is convex.
func Prism(poly [][2]float32, depth float32, caps [][]int) rl.Mesh {
	var pos, nor []float32
	tri := func(a, b, c, n rl.Vector3) {
		// Wind counter-clockwise as seen from the side the normal points to.
		if rl.Vector3DotProduct(rl.Vector3CrossProduct(rl.Vector3Subtract(b, a), rl.Vector3Subtract(c, a)), n) < 0 {
			b, c = c, b
		}
		for _, v := range []rl.Vector3{a, b, c} {
			pos = append(pos, v.X, v.Y, v.Z)
			nor = append(nor, n.X, n.Y, n.Z)
		}
	}
	z := depth / 2
	at := func(i int, z float32) rl.Vector3 { return rl.NewVector3(poly[i][0], poly[i][1], z) }
	for i := range poly {
		j := (i + 1) % len(poly)
		n := rl.Vector3Normalize(rl.NewVector3(poly[j][1]-poly[i][1], poly[i][0]-poly[j][0], 0))
		tri(at(i, -z), at(j, -z), at(j, z), n)
		tri(at(i, -z), at(j, z), at(i, z), n)
	}
	if caps == nil {
		all := make([]int, len(poly))
		for i := range all {
			all[i] = i
		}
		caps = [][]int{all}
	}
	for _, piece := range caps {
		for k := 1; k+1 < len(piece); k++ {
			for _, side := range []float32{-1, 1} {
				tri(at(piece[0], side*z), at(piece[k], side*z), at(piece[k+1], side*z), rl.NewVector3(0, 0, side))
			}
		}
	}
	uv := make([]float32, len(pos)/3*2)
	meshData = append(meshData, pos, nor, uv)
	m := rl.Mesh{VertexCount: int32(len(pos) / 3), TriangleCount: int32(len(pos) / 9)}
	m.Vertices, m.Normals, m.Texcoords = &pos[0], &nor[0], &uv[0]
	rl.UploadMesh(&m, false)
	return m
}
