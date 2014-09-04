// +build !windows
//fork from gocode

package main

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
)

// Full path of the current executable
func get_executable_filename() string {
	// try readlink first
	path, err := os.Readlink("/proc/self/exe")
	if err == nil {
		return path
	}
	// use argv[0]
	path = os.Args[0]
	if !filepath.IsAbs(path) {
		cwd, _ := os.Getwd()
		path = filepath.Join(cwd, path)
	}
	if file_exists(path) {
		return path
	}
	// Fallback : use "gocode" and assume we are in the PATH...
	path, err = exec.LookPath("gofn")
	if err == nil {
		return path
	}
	return ""
}

// config location

func config_dir() string {
	return filepath.Join(home_dir(), "gofn")
}

func config_file() string {
	return filepath.Join(home_dir(), "gofn", "config.json")
}
