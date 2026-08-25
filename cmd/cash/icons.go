package main

import (
	_ "embed"

	"c.ash/internal/domain"
)

//go:embed icons/light.png
var lightIcon []byte

//go:embed icons/dark.png
var darkIcon []byte

//go:embed icons/gothic.png
var gothicIcon []byte

func nativeIcon(theme domain.Theme) []byte {
	switch theme {
	case domain.ThemeDark:
		return darkIcon
	case domain.ThemeGothic:
		return gothicIcon
	default:
		return lightIcon
	}
}
