package core

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

// CleanString trims all leading and trailing whitespace in `s` and optionally lowers it.
func CleanString(s string, lower ...bool) string {
	s = strings.TrimSpace(s)
	if len(lower) > 0 && lower[0] {
		return strings.ToLower(s)
	}
	return s
}

// Getwd tries to find the project root "backend".
// go-test changes the working directory to the test package being run during tests... this breaks our code...
// see: https://stackoverflow.com/questions/23847003/golang-tests-and-working-directory
// this is a temporary fix for now :(
func Getwd() string {
	root := "backend"
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	currDir := wd
	for {
		if fi, err := os.Stat(currDir); err == nil {
			dirBase := filepath.Base(currDir)
			if dirBase == root && fi.IsDir() {
				return currDir
			}
		}
		newDir := filepath.Dir(currDir)
		if newDir == string(os.PathSeparator) || newDir == currDir {
			log.Fatal("project root not found")
		}
		currDir = newDir
	}
}
