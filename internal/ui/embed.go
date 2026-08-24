package ui

import "embed"

//go:embed static/index.html
var indexHTML []byte

//go:embed static/app.css
var appCSS []byte

//go:embed static/app.js
var appJS []byte

//go:embed static/forge.js
var forgeJS []byte

//go:embed static/three.min.js
var threeJS []byte

//go:embed static/afp-mark.png
var afpMarkPNG []byte

//go:embed static/fonts/*.woff2
var fontFS embed.FS
