package e2e_test

import (
	"os"
	"testing"
)

// tempDirs tracks temp directories created during tests for cleanup.
var tempDirs []string

func trackTempDir(dir string) {
	tempDirs = append(tempDirs, dir)
}

func TestMain(m *testing.M) {
	code := m.Run()
	for _, dir := range tempDirs {
		os.RemoveAll(dir)
	}
	os.Exit(code)
}
