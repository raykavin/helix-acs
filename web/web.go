// Package web embeds the static web UI files served by the API server.
package web

import "embed"

//go:embed index.html app.js style.css
var FS embed.FS
