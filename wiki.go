package main

import (
	"bytes"
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/inhies/go-bytesize" // Do we really need this?
	"io/ioutil"
	"os"
	"os/exec"
	fpath "path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

type WikiTrackData struct {
	Duration                 string
	FolderName               string
	Index                    int
	HasAlternateLeadVocalist bool
	Name                     string
	CD                       int
}

type WikiAlbumData struct {
	Notes      string
	FolderName string
	Tracks     []WikiTrackData
	Duration   string
	Lineage    string
	Size       string
	SampleRate string
	BPS        string
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

	removeFile(wikifile, deleteMode)
	if deleteMode {
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

	size, err := getDirectorySize(filepath)
	if err != nil {
		fmt.Println("failed to get directory size")
		fmt.Println(err)
		return
	}
	b := bytesize.New(size)
	parsedData.Size = b.String()

	directoryContents, err := ioutil.ReadDir(filepath)
	if err != nil {
		fmt.Println("failed to get directory contents")
		fmt.Println(err)
		return
	}

	for _, file := range directoryContents {
		if file.IsDir() || fpath.Ext(file.Name()) != ".flac" {
			continue
		}

		data, err := exec.Command(
			"metaflac",
			"--show-sample-rate",
			"--show-bps", // update numbers below
			fpath.Join(filepath, file.Name()),
		).Output()

		if err != nil {
			fmt.Println("metaflac returned an invalid response for sample-rate/bps")
			if data != nil {
				fmt.Println(data)
			}
			panic(err)
		}

		lines := strings.Split(string(data), "\n")
		if len(lines) != 1+2 {
			// Update '2' below and above
			fmt.Println("expected 2 metaflac lines, got", len(lines)-1)
			return
		}

		for i, line := range lines {
			line := strings.TrimSpace(line)

			if i == 0 {
				rateFloat, err := strconv.ParseFloat(line, 32)
				if err != nil {
					fmt.Print("could not convert sample-rate to float")
					fmt.Println(err)
					return
				}
				parsedData.SampleRate = strconv.FormatFloat(rateFloat/1000, 'f', -1, 32) + "KHz"
			} else if i == 1 {
				parsedData.BPS = line
			}
		}
		break
	}

	tracks := make([]WikiTrackData, 0)

	var lastTrack WikiTrackData
	var currentTrackNumber int = 0

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

				str := strings.TrimSpace(track)
				f := strings.Index(str, "[")
				l := strings.Index(str, "]")
				trackData.Duration = str[f+1 : l]

				number := str[:f-2]
				separator := strings.Index(number, ".")
				if separator != -1 {
					// This bit only uses the "path" library
					// because URL's only use forward slash
					cdStr := number[:separator]

					cdNumber, err := strconv.Atoi(cdStr)
					if err != nil {
						panic(err.Error())
					}
					trackData.FolderName = "CD" + cdStr
					trackData.CD = cdNumber

					if (index != 0) && (lastTrack.CD != cdNumber) {
						currentTrackNumber = 0
					}
				} else {
					// Here lies incomplete support for "Bonus." tracks
					// you need to add support for using filenames instead
					// of tracknumbers for bonus tracks

					// _, err := strconv.Atoi(number)
					// if err != nil {
					// 	trackData.FolderName = number
					// 	trackData.CD = -1

					// 	if lastTrack.FolderName != number {
					// 		currentTrackNumber = 0
					// 	}
					// }
					continue
				}

				name := strings.TrimSpace(str[l+1:])
				nameWithoutSuffix := strings.TrimSuffix(name, " (*)")
				trackData.HasAlternateLeadVocalist = name != nameWithoutSuffix
				trackData.Name = nameWithoutSuffix

				currentTrackNumber += 1
				trackData.Index = currentTrackNumber

				lastTrack = trackData
				tracks = append(tracks, trackData)
			}
		case 4:
			parsedData.Duration = field
		}
	}
	parsedData.Tracks = tracks

	wikiout := createFile(wikifile)
	defer wikiout.Close()

	if wikiout != nil {
		err = wikiTemplate.Execute(wikiout, parsedData)
		if err != nil {
			fmt.Println("could not insert data into template!")
			fmt.Println(err)
			return
		}

		fmt.Println("success!")
	}
}
