//go:build !windows

package main

import _ "embed"

//go:embed data/favicon-32x32.png
var trayIcon []byte
