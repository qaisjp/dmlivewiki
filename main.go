package main

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Println("Syntax: dmlivewiki_checksum <path> [-y]")
		return
	}

	filepath := os.Args[1]

	// Ignore error, it returns false
	// even if it doesn't exist
	isDirectory, _ := isDirectory(filepath)
	if !isDirectory {
		fmt.Println("Error: target is not a directory")
		return
	}

	if !shouldContinue(filepath) {
		return
	}

	files, _ := ioutil.ReadDir(filepath)
	for _, file := range files {
		if file.IsDir() {
			ffp := createFile(filepath, file.Name(), "ffp")
			processDirectory(path.Join(filepath, file.Name()), 1, ffp, "ffp")
			ffp.Close()

			md5 := createFile(filepath, file.Name(), "md5")
			processDirectory(path.Join(filepath, file.Name()), 1, md5, "md5")
			md5.Close()
		}
	}
}

func shouldContinue(filepath string) bool {
	// Ask to continue or just process?
	// Hacky!
	if len(os.Args) > 2 {
		if os.Args[2] == "-y" {
			return true
		}
	}

	fmt.Println("The following filepath will be processed: ", filepath)
	fmt.Print("Continue? (y/n): ")
	text := ""
	fmt.Scanln(&text)
	if text != "y" {
		return false
	}
	return true
}

func processDirectory(filepath string, depth int, out *os.File, mode string) {
	files, _ := ioutil.ReadDir(filepath)
	if len(files) == 0 {
		if mode == "ffp" {
			fmt.Println("Empty folder found:", filepath)
		}
		return
	}

	var parser func(string, string, int) string
	if mode == "ffp" {
		parser = ffpParse
	} else if mode == "md5" {
		parser = md5Parse
	}

	for _, file := range files {
		name := file.Name()

		if file.IsDir() {
			processDirectory(path.Join(filepath, name), depth+1, out, mode)
		} else if (path.Ext(name) != ".md5") && !file.IsDir() {
			if result := parser(filepath, name, depth); result != "" {
				out.WriteString(result)
			}
		}
	}
}

func md5Parse(filepath string, name string, depth int) string {
	data, err := ioutil.ReadFile(path.Join(filepath, name))

	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%x *%s%s\n", md5.Sum(data), getLastPathComponents(filepath, depth), name)
}

func ffpParse(filepath string, name string, depth int) string {
	if path.Ext(name) != ".flac" {
		return ""
	}

	data, err := exec.Command(
		"metaflac",
		"--show-md5sum",
		path.Join(filepath, name),
	).Output()

	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%s%s:%s", getLastPathComponents(filepath, depth), name, data)
}
