package main

import (
	"bytes"
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/inhies/go-bytesize" // Do we really need this?
	"io/ioutil"
	"os"
	"os/exec"
	upath "path"
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
	LinePrefix               string
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
		generateWikifile(filepath, fileInfo.Name(), regex, wikiTemplate, c.GlobalBool("delete"), "")
		return
	}

	// Create the wikifiles folder path
	wikifiles := fpath.Join(filepath, "__wikifiles")

	// MkdirAll is used instead of Mkdir because this function
	// doesn't error if the folder already exists
	err = os.MkdirAll(wikifiles, os.ModeDir)
	if err != nil {
		fmt.Println("Internal error creating __wikifiles folder")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	files, _ := ioutil.ReadDir(filepath)
	for _, file := range files {
		if file.IsDir() {
			name := file.Name()
			if name != "__wikifiles" {
				generateWikifile(fpath.Join(filepath, name), name, regex, wikiTemplate, c.GlobalBool("delete"), wikifiles)
			}
		}
	}
}

func wikiGetInfoFromFlac(filepath string, parsedData *WikiAlbumData) bool {
	directoryContents, err := ioutil.ReadDir(filepath)
	if err != nil {
		fmt.Println("failed to get directory contents")
		fmt.Println(err)
		return false
	}

	for _, file := range directoryContents {
		if file.IsDir() {
			if wikiGetInfoFromFlac(fpath.Join(filepath, file.Name()), parsedData) {
				return true
			}
		} else if fpath.Ext(file.Name()) == ".flac" {
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
				continue
			}

			failure := false
			for i, line := range lines {
				line := strings.TrimSpace(line)

				if i == 0 {
					rateFloat, err := strconv.ParseFloat(line, 32)
					if err != nil {
						fmt.Print("could not convert sample-rate to float")
						fmt.Println(err)
						failure = true
						break
					}
					parsedData.SampleRate = strconv.FormatFloat(rateFloat/1000, 'f', -1, 32) + "KHz"
				} else if i == 1 {
					parsedData.BPS = line
				}
			}

			if !failure {
				return true
			}
		}
	}

	fmt.Println("failed to find file for sampling info")
	return false
}

func generateWikifile(filepath string, foldername string, regex *regexp.Regexp, wikiTemplate *template.Template, deleteMode bool, outBasepath string) {
	basepath := fpath.Join(filepath, foldername)
	infofile := basepath + ".txt"
	wikifile := basepath + ".wiki"

	if outBasepath != "" {
		wikifile = fpath.Join(outBasepath, foldername) + ".wiki"
	}

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

	if !wikiGetInfoFromFlac(filepath, &parsedData) {
		return
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
			lineage := ""
			for _, item := range strings.Split(field, "\n") {
				lineage += "*" + strings.TrimSpace(item) + "\r\n"
			}
			parsedData.Lineage = lineage
		case 2:
			parsedData.Notes = field
		case 3:
			// parse tracks
			for _, track := range strings.Split(field, "\n") {
				var trackData WikiTrackData
				trackData.FolderName = foldername
				trackData.LinePrefix = "#"

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
					trackData.FolderName = upath.Join(foldername, "CD"+cdStr)
					trackData.CD = cdNumber

					if lastTrack.CD != cdNumber {
						currentTrackNumber = 0
						trackData.LinePrefix = fmt.Sprintf("\r\nCD%d:\r\n%s", cdNumber, trackData.LinePrefix)
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
					// continue
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
