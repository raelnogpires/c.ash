package main

import (
	"bytes"
	"testing"

	"c.ash/internal/domain"
)

func TestNativeIcon_FollowsTheme(t *testing.T) {
	for _, test := range []struct {
		name  string
		theme domain.Theme
		want  []byte
	}{
		{name: "light", theme: domain.ThemeLight, want: lightIcon},
		{name: "dark", theme: domain.ThemeDark, want: darkIcon},
		{name: "gothic", theme: domain.ThemeGothic, want: gothicIcon},
		{name: "fallback", theme: "", want: lightIcon},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := nativeIcon(test.theme)
			if len(got) == 0 || !bytes.Equal(got, test.want) {
				t.Fatal("nativeIcon returned the wrong embedded asset")
			}
		})
	}
}
