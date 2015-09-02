package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	fpath "path/filepath"
	"strings"
)

func wikiescape(s string) string {
	return url.QueryEscape(strings.Replace(s, " ", "_", -1))
}

func createFile(filename string) *os.File {
	f, err := os.Create(filename)
	if err != nil {
		panic(f)
	}
	return f
}

func removeFile(filename string, log bool) {
	if log {
		fmt.Printf("Removing %s...", filename)
	}

	err := os.Remove(filename)
	if err != nil {
		if os.IsNotExist(err) {
			if log {
				fmt.Println(" does not exist!")
			}
			return
		}
		panic(err)
	}
	if log {
		fmt.Println(" success!")
	}
}

func isDirectory(filepath string) (bool, os.FileInfo, error) {
	fileInfo, err := os.Stat(filepath)
	if err != nil {
		return false, nil, err
	}
	return fileInfo.IsDir(), fileInfo, err
}

func getDirectorySize(filepath string) (float64, error) {
	var size float64
	contents, err := ioutil.ReadDir(filepath)
	if err != nil {
		return -1, err
	}

	for _, file := range contents {
		if file.IsDir() {
			subsize, err := getDirectorySize(fpath.Join(filepath, file.Name()))
			if err != nil {
				return -1, err
			}
			size += subsize
		} else {
			size += float64(file.Size())
		}
	}

	return size, nil
}

func getLastPathComponents(filepath string, depth int) (absPath string) {
	for i := 1; i < depth; i++ {
		absPath = path.Base(filepath) + "\\" + absPath
		filepath = path.Dir(filepath)
	}
	return
}

func checkFilepathArgument(c *cli.Context) (os.FileInfo, string) {
	if len(c.Args()) != 1 {
		cli.ShowSubcommandHelp(c)
		return nil, ""
	}

	filepath := c.Args()[0]
	return getFileOfType(filepath, true, "target")
}

func getFileOfType(filepath string, wantDirectory bool, target string) (os.FileInfo, string) {
	isDirectory, fileInfo, _ := isDirectory(filepath)
	if (fileInfo == nil) || (isDirectory != wantDirectory) {
		if wantDirectory {
			fmt.Println("Error:", target, "is not a directory")
		} else {
			fmt.Println("Error:", target, "is not a file")
		}
		return nil, ""
	}
	return fileInfo, path.Clean(filepath)
}

func shouldContinue(c *cli.Context) bool {
	// Ask to continue or just process?
	if c.GlobalBool("force") {
		fmt.Print("\n")
		return true
	}

	fmt.Print("Continue? (y/n): ")
	text := ""
	fmt.Scanln(&text)
	fmt.Print("\n")

	if text != "y" {
		return false
	}
	return true
}

func notifyDeleteMode(c *cli.Context) {
	if c.GlobalBool("delete") {
		fmt.Print("You are running in **DELETE MODE** - data will be permanently lost")
	}
}

func ifTrimPrefix(s, prefix string) string {
	if !strings.HasPrefix(s, prefix) {
		panic(fmt.Sprintf("Expected prefix %s, but line is: %s", prefix, s))
		// TODO: see above error handling
	}
	return strings.TrimPrefix(s, prefix)
}
