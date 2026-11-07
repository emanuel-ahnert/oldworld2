//go:build bundle

package assets

import _ "embed"

// bundle is the zip made by tools/release/bundle.py.
//
//go:embed bundle.zip
var bundle string
