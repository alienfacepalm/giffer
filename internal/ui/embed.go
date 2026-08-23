package ui

import _ "embed"

//go:embed static/index.html
var indexHTML []byte

//go:embed static/app.css
var appCSS []byte

//go:embed static/app.js
var appJS []byte
