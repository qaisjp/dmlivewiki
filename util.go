package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"net/url"
	"os"
	"path"
	"strings"
)

func wikiescape(s string) string {
	return url.QueryEscape(strings.Replace(s, " ", "_", -1))
}

func createFile(filename string) *os.File {
	fmt.Println("Creating", filename+"...")
	f, err := os.Create(filename)
	if err != nil {
		panic(f)
	}
	return f
}

func removeFile(filename string) {
	fmt.Printf("Removing %s...", filename)

	err := os.Remove(filename)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println(" does not exist!")
			return
		}
		panic(err)
	}
	fmt.Println(" success!")
}

func isDirectory(filepath string) (bool, os.FileInfo, error) {
	fileInfo, err := os.Stat(filepath)
	if err != nil {
		return false, nil, err
	}
	return fileInfo.IsDir(), fileInfo, err
}

func getLastPathComponents(filepath string, depth int) (absPath string) {
	for i := 1; i < depth; i++ {
		absPath = path.Base(filepath) + "\\" + absPath
		filepath = path.Dir(filepath)
	}
	return
}

func checkFilepathArgument(c *cli.Context) (string, os.FileInfo) {
	if len(c.Args()) != 1 {
		cli.ShowSubcommandHelp(c)
		return "", nil
	}

	filepath := c.Args()[0]

	// Ignore error, it returns false
	// even if it doesn't exist
	isDirectory, fileInfo, _ := isDirectory(filepath)
	if !isDirectory {
		fmt.Println("Error: target is not a directory")
		return "", nil
	}

	return path.Clean(filepath), fileInfo
}

func shouldContinue(c *cli.Context) bool {
	// Ask to continue or just process?
	if c.GlobalBool("force") {
		fmt.Print("\n")
		return true
	}

	if c.GlobalBool("delete") {
		fmt.Print("[Delete mode] ")
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

func ifTrimPrefix(s, prefix string) string {
	if !strings.HasPrefix(s, prefix) {
		panic(fmt.Sprintf("Expected prefix %s, but line is: ", prefix, s))
		// TODO: see above error handling
	}
	return strings.TrimPrefix(s, prefix)
}
