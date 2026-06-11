//go:build !unix

package tui

// cellPixelSize is unavailable off unix; callers fall back to a 1:2 cell.
var cellPixelSize = func() (w, h int) { return 0, 0 }
