//go:build !linux

package main

import "github.com/wailsapp/wails/v2/pkg/options/linux"

func configureWebView() {}

func platformOptions([]byte) *linux.Options { return nil }
func setPlatformIcon([]byte)                {}
