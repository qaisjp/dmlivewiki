package main

import (
	"fmt"
	"io/ioutil"
	"os"
	fpath "path/filepath"
	"regexp"
	"strings"

	"github.com/qaisjp/dmlivewiki/util"
	"gopkg.in/urfave/cli.v1"
)

func findWikifiles(c *cli.Context) {
	fileInfo, filepath := util.CheckFilepathArgument(c)
	if fileInfo == nil {
		return
	}

	mode := "batch"
	if c.GlobalBool("single") {
		mode = "single"
	}

	fmt.Printf("The following filepath (%s mode) will be processed: %s\n", mode, filepath)
	util.NotifyDeleteMode(c)

	if !util.ShouldContinue(c) {
		return
	}

	wikiRegex = regexp.MustCompile(wikiRegexText)

	if mode == "single" {
		findWikifile(filepath, fileInfo.Name())
		return
	}

	files, _ := ioutil.ReadDir(filepath)
	for _, file := range files {
		if file.IsDir() {
			name := file.Name()
			if name != "__wikifiles" {
				findWikifile(fpath.Join(filepath, name), name)
			}
		}
	}
}

func findWikifile(filepath string, foldername string) {
	infofile := fpath.Join(filepath, foldername+".txt")

	infobytes, err := ioutil.ReadFile(infofile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No infofile for", infofile)
		} else {
			fmt.Printf("error in %s (%s) \n", infofile, err.Error())
		}
		return
	}

	matches := wikiRegex.FindSubmatch(infobytes)
	if len(matches) != 1+wikiRegex.NumSubexp() {
		// (entire string itself)+(capture groups)
		fmt.Printf("parse failure, expected %d capturing groups!\n", 1+wikiRegex.NumSubexp())
		return
	}

	if strings.TrimSpace(string(matches[3])) == "" {
		fmt.Println("Notes unfilled for", infofile)
	}

}
