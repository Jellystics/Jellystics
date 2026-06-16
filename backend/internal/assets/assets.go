package assets

import "embed"

// Web holds the compiled frontend SPA.
// Populated at build time by copying frontend/dist → backend-v2/internal/assets/web/
//
//go:embed web
var Web embed.FS
