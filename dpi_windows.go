package main

import "golang.org/x/sys/windows"

func init() {
	// DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 (-4) gives crisp rendering
	// on high-DPI and multi-monitor setups (requires Windows 10 1703+).
	// Falls back gracefully on older Windows — the call simply fails silently.
	user32 := windows.NewLazySystemDLL("user32.dll")
	user32.NewProc("SetProcessDpiAwarenessContext").Call(^uintptr(3)) // -4 as uintptr
}
