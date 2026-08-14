package assets

import "embed"

// FS contains the immutable templates and browser assets shipped in the image.
//
//go:embed templates static
var FS embed.FS
