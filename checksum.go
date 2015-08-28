package main

import (
	"crypto/md5"
	"fmt"
	"github.com/codegangsta/cli"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
)

func performChecksum(c *cli.Context) {
	filepath, fileInfo := checkFilepathArgument(c)
	if filepath == "" {
		return
	}

	if !shouldContinue(c, filepath) {
		return
	}

	if c.Bool("single") {
		processPath(path.Dir(filepath), fileInfo.Name(), c.GlobalBool("delete"))
		return
	}

	files, _ := ioutil.ReadDir(filepath)
	for _, file := range files {
		if file.IsDir() {
			processPath(filepath, file.Name(), c.GlobalBool("delete"))
		}
	}
}

func processPath(filepath string, name string, deleteMode bool) {
	directory := path.Join(filepath, name)
	filename := path.Join(directory, name+".")

	if deleteMode {
		removeFile(filename + "ffp")
		removeFile(filename + "md5")
		return
	}

	ffp := createFile(filename + "ffp")
	processDirectory(directory, 1, ffp, "ffp")
	ffp.Close()

	md5 := createFile(filename + "md5")
	processDirectory(directory, 1, md5, "md5")
	md5.Close()
}

func shouldContinue(c *cli.Context, filepath string) bool {
	// Ask to continue or just process?
	if c.GlobalBool("force") {
		return true
	}

	mode := "batch"
	if c.Bool("single") {
		mode = "single"
	}

	fmt.Printf("The following filepath (%s mode) will be processed: %s\n", mode, filepath)
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