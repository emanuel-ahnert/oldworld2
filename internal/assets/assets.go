// Package assets loads image files from the assets directory.
package assets

import (
	"fmt"
	"image"
	_ "image/png" // register the PNG decoder
	"os"
	"path/filepath"
)

// Asset paths, relative to the assets directory.
const (
	// Font is the 8×8 bitmap font sheet for all UI text.
	Font = "generated/ui/font8x8.png" // tools/font/makefont.py
)

// Path returns the file path of an asset.
func Path(dir, rel string) string { return filepath.Join(dir, rel) }

// LoadImage decodes an image relative to the assets directory.
func LoadImage(dir, rel string) (image.Image, error) {
	f, err := os.Open(Path(dir, rel))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", rel, err)
	}
	return img, nil
}

// FindDir returns the assets directory: the given path if set, otherwise the assets built into
// a release build, otherwise "assets" in the working directory or next to the executable.
func FindDir(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if bundle != "" {
		return unpackBundle()
	}
	candidates := []string{"assets"}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "assets"))
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			return c, nil
		}
	}
	return "", fmt.Errorf("assets directory not found (tried %v); use -assets", candidates)
}
