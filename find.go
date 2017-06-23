package main

import (
	"fmt"
	"io/ioutil"
	"os"
	fpath "path/filepath"
	"regexp"
	"strings"

	"gopkg.in/urfave/cli.v1"
)

func findWikifiles(c *cli.Context) {
	fileInfo, filepath := checkFilepathArgument(c)
	if fileInfo == nil {
		return
	}

	mode := "batch"
	if c.GlobalBool("single") {
		mode = "single"
	}

	fmt.Printf("The following filepath (%s mode) will be processed: %s\n", mode, filepath)
	notifyDeleteMode(c)

	if !shouldContinue(c) {
		return
	}

	regex, err := regexp.Compile(wikiRegex)
	if err != nil {
		fmt.Println("Internal error - wiki regex could not be compiled!")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if mode == "single" {
		findWikifile(filepath, fileInfo.Name(), regex)
		return
	}

	files, _ := ioutil.ReadDir(filepath)
	for _, file := range files {
		if file.IsDir() {
			name := file.Name()
			if name != "__wikifiles" {
				findWikifile(fpath.Join(filepath, name), name, regex)
			}
		}
	}
}

func findWikifile(filepath string, foldername string, regex *regexp.Regexp) {
	infofile := fpath.Join(filepath, foldername+".txt")

	infobytes, err := ioutil.ReadFile(infofile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("infofile doesn't exist")
		} else {
			fmt.Printf("error (%s)\n", err.Error())
		}
		return
	}

	matches := regex.FindSubmatch(infobytes)
	if len(matches) != 1+regex.NumSubexp() {
		// (entire string itself)+(capture groups)
		fmt.Printf("parse failure, expected %d capturing groups!\n", 1+regex.NumSubexp())
		return
	}

	if strings.TrimSpace(string(matches[3])) == "" {
		fmt.Println("Notes unfilled for", infofile)
	}

}