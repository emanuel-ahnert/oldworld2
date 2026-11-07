package assets

import (
	"archive/zip"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// unpackBundle unpacks the assets built into a release build (tools/release) to the user's
// cache folder, once per version, and returns the assets directory. It changes to that folder
// and returns a relative path: raylib opens files with fopen, which on Windows can't open paths
// with non-ASCII characters, as in many a user name.
func unpackBundle() (string, error) {
	zr, err := zip.NewReader(strings.NewReader(bundle), int64(len(bundle)))
	if err != nil {
		return "", fmt.Errorf("built-in assets: %w", err)
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	// Named after the contents, so that a new version gets its own copy.
	h := crc32.NewIEEE()
	for _, f := range zr.File {
		fmt.Fprintf(h, "%s %d\n", f.Name, f.CRC32)
	}
	base := filepath.Join(cache, "oldworld2")
	root := filepath.Join(base, fmt.Sprintf("game-%08x", h.Sum32()))
	if _, err := os.Stat(root); err != nil {
		// Into a temporary folder first: an interrupted first start must not leave half a copy.
		tmp := root + ".tmp"
		if err := os.RemoveAll(tmp); err != nil {
			return "", err
		}
		for _, f := range zr.File {
			if err := unpackFile(f, filepath.Join(tmp, filepath.FromSlash(f.Name))); err != nil {
				return "", fmt.Errorf("unpack assets: %w", err)
			}
		}
		if err := os.Rename(tmp, root); err != nil {
			return "", fmt.Errorf("unpack assets: %w", err)
		}
		// The copies of older versions aren't needed any more.
		old, _ := filepath.Glob(filepath.Join(base, "game-*"))
		for _, p := range old {
			if p != root {
				os.RemoveAll(p)
			}
		}
	}
	if err := os.Chdir(root); err != nil {
		return "", err
	}
	return "assets", nil
}

func unpackFile(f *zip.File, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	in, err := f.Open()
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
