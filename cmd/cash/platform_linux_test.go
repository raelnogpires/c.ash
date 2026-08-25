//go:build linux

package main

import (
	"os"
	"testing"
)

func TestConfigureWebView_DefaultsToReliableRenderer(t *testing.T) {
	const key = "WEBKIT_DISABLE_DMABUF_RENDERER"
	previous, existed := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(key, previous)
		} else {
			_ = os.Unsetenv(key)
		}
	})

	configureWebView()
	if got := os.Getenv(key); got != "1" {
		t.Fatalf("%s=%q, want 1", key, got)
	}
}

func TestConfigureWebView_RespectsExplicitSetting(t *testing.T) {
	t.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "0")
	configureWebView()
	if got := os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER"); got != "0" {
		t.Fatalf("WEBKIT_DISABLE_DMABUF_RENDERER=%q, want 0", got)
	}
}

func TestPlatformOptions_UsesSelectedIcon(t *testing.T) {
	icon := []byte("theme icon")
	got := platformOptions(icon)
	if got == nil || string(got.Icon) != string(icon) || got.ProgramName != "cash" {
		t.Fatalf("platformOptions() = %#v", got)
	}
}
