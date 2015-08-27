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

func main() {
	app := cli.NewApp()
	app.Name = "dmlivewiki_checksum"
	app.Usage = "" // todo

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "yes, y",
			Usage: "skip the confirmation input",
		},
		cli.BoolFlag{
			Name:  "single, s",
			Usage: "parse the directory given, not the subdirectories",
		},
	}

	app.Action = mainAction
	app.Version = "1.0.2"

	app.Run(os.Args)
}

func mainAction(c *cli.Context) {
	if len(c.Args()) != 1 {
		fmt.Println("Syntax: dmlivewiki_checksum [options] <path>")
		return
	}

	filepath := c.Args()[0]

	// Ignore error, it returns false
	// even if it doesn't exist
	isDirectory, fileInfo, _ := isDirectory(filepath)
	if !isDirectory {
		fmt.Println("Error: target is not a directory")
		return
	}

	if !shouldContinue(c, filepath) {
		return
	}

	if c.Bool("single") {
		processPath(path.Dir(filepath), fileInfo.Name())
		return
	}

	files, _ := ioutil.ReadDir(filepath)
	for _, file := range files {
		if file.IsDir() {
			processPath(filepath, file.Name())
		}
	}
}

func processPath(filepath string, name string) {
	directory := path.Join(filepath, name)

	ffp := createFile(filepath, name, "ffp")
	processDirectory(directory, 1, ffp, "ffp")
	ffp.Close()

	md5 := createFile(filepath, name, "md5")
	processDirectory(directory, 1, md5, "md5")
	md5.Close()
}

func shouldContinue(c *cli.Context, filepath string) bool {
	// Ask to continue or just process?
	// Hacky!
	if c.Bool("yes") {
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
