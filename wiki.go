package main

import (
	"bytes"
	"fmt"
	"github.com/codegangsta/cli"
	"io/ioutil"
	"os"
	fpath "path/filepath"
	"regexp"
	"strings"
	"text/template"
)

type WikiTrackData struct {
	Duration                 string
	FolderName               string
	Index                    int
	HasAlternateLeadVocalist bool
	Name                     string
}

type WikiAlbumData struct {
	Notes      string
	FolderName string
	Tracks     []WikiTrackData
	Duration   string
	Lineage    string
	Size       string
}

func generateWikifiles(c *cli.Context) {
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

	wikiTemplate, err := template.New("wiki").Parse(
		// Stupid windows
		strings.Replace(wikiTemplate, "\n", "\r\n", -1),
	)
	if err != nil {
		fmt.Println("Internal error - wiki template could not be parsed!")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if mode == "single" {
		generateWikifile(filepath, fileInfo.Name(), regex, wikiTemplate, c.GlobalBool("delete"))
		return
	}

	files, _ := ioutil.ReadDir(filepath)
	for _, file := range files {
		if file.IsDir() {
			name := file.Name()
			generateWikifile(fpath.Join(filepath, name), name, regex, wikiTemplate, c.GlobalBool("delete"))
		}
	}
}

func generateWikifile(filepath string, foldername string, regex *regexp.Regexp, wikiTemplate *template.Template, deleteMode bool) {
	basepath := fpath.Join(filepath, foldername)
	infofile := basepath + ".txt"
	wikifile := basepath + ".wiki"

	if deleteMode {
		removeFile(wikifile)
		return
	}

	fmt.Printf("Generating from %s... ", infofile)
	infobytes, err := ioutil.ReadFile(infofile)
	if err != nil {
		reason := "infofile doesn't exist"
		if !os.IsNotExist(err) {
			reason = err.Error()
		}

		fmt.Printf("could not open file (%s)\n", reason)
		return
	}

	matches := regex.FindSubmatch(infobytes)
	if len(matches) != 1+4 {
		// 1 (entire string itself) + 4 (capture groups)
		fmt.Println("expected 7 capturing groups, failure parsing!")
		return
	}

	var parsedData WikiAlbumData
	parsedData.FolderName = foldername

	tracks := make([]WikiTrackData, 0)
	for i, field := range matches {
		if i == 0 {
			// The first one is just itself
			// Note: learn why it's just the input string
			continue
		}

		field := string(bytes.TrimSpace(field))

		switch i {
		case 1:
			parsedData.Lineage = field
		case 2:
			parsedData.Notes = field
		case 3:
			// parse tracks
			for index, track := range strings.Split(field, "\n") {
				var trackData WikiTrackData
				trackData.Index = index + 1

				str := strings.TrimSpace(track)
				f := strings.Index(str, "[")
				l := strings.Index(str, "]")
				trackData.Duration = str[f+1 : l]

				name := strings.TrimSpace(str[l+1:])
				nameWithoutSuffix := strings.TrimSuffix(name, " (*)")
				trackData.HasAlternateLeadVocalist = name != nameWithoutSuffix
				trackData.Name = nameWithoutSuffix
				trackData.FolderName = foldername
				tracks = append(tracks, trackData)
			}
		case 4:
			parsedData.Duration = field
		}
	}

	parsedData.Tracks = tracks
	parsedData.Size = "TO DO ME UP"

	wikiout := createFile(wikifile)
	defer wikiout.Close()

	err = wikiTemplate.Execute(wikiout, parsedData)
	if err != nil {
		fmt.Println("could not insert data into template!")
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("success!")
}
