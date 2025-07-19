package main

import "embed"

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/js/*.js
var staticFS embed.FS