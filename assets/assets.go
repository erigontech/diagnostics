package assets

import "embed"

//go:embed template
var Templates embed.FS

//go:embed script
var Scripts embed.FS
