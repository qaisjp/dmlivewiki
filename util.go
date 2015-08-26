package main

import (
	"fmt"
	"os"
	"path"
)

func createFile(filepath string, subdir string, ext string) *os.File {
	filename := path.Join(filepath, subdir, subdir+"."+ext)

	fmt.Println("Creating", filename+"...")
	f, err := os.Create(filename)
	if err != nil {
		panic(f)
	}
	return f
}

func isDirectory(filepath string) (bool, error) {
	fileInfo, err := os.Stat(filepath)
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), err
}

func getLastPathComponents(filepath string, depth int) (absPath string) {
	for i := 1; i < depth; i++ {
		absPath = path.Base(filepath) + "\\" + absPath
		filepath = path.Dir(filepath)
	}
	return
}
