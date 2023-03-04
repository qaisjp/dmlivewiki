package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	upath "path"
	fpath "path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/inhies/go-bytesize" // Do we really need this?
	"github.com/qaisjp/dmlivewiki/util"
	"gopkg.in/urfave/cli.v1"
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

var bracketRegex *regexp.Regexp
var wikiRegex *regexp.Regexp

func generateWikifiles(c *cli.Context) {
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
	bracketRegex = regexp.MustCompile(`".*?"`)

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
		generateWikifile(filepath, fileInfo.Name(), wikiTemplate, c.GlobalBool("delete"), "")
		return
	}

	// Create the wikifiles folder path
	wikifiles := fpath.Join(filepath, "__wikifiles")

	// MkdirAll is used instead of Mkdir because this function
	// doesn't error if the folder already exists
	err = os.MkdirAll(wikifiles, os.ModePerm)
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
				generateWikifile(fpath.Join(filepath, name), name, wikiTemplate, c.GlobalBool("delete"), wikifiles)
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
				metaflacPath,
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

func generateWikifile(filepath string, foldername string, wikiTemplate *template.Template, deleteMode bool, outBasepath string) {
	basepath := fpath.Join(filepath, foldername)
	infofile := basepath + ".txt"

	// This is updated later
	wikifile := filepath
	if outBasepath != "" {
		wikifile = outBasepath
	}

	if deleteMode {
		fmt.Printf("Deleting wiki from %s...", infofile)
	} else {
		fmt.Printf("Generating from %s... ", infofile)
	}

	infobytes, err := ioutil.ReadFile(infofile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("infofile doesn't exist")
		} else {
			fmt.Printf("error (%s)\n", err.Error())
		}
		return
	}

	matches := wikiRegex.FindSubmatch(infobytes)
	if len(matches) != 1+wikiRegex.NumSubexp() {
		// (entire string itself)+(capture groups)
		fmt.Printf("parse failure, expected %d capturing groups!\n", 1+wikiRegex.NumSubexp())
		return
	}

	var parsedData WikiAlbumData
	parsedData.FolderName = foldername

	size, err := util.GetDirectorySite(filepath)
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

	var tracks []WikiTrackData
	var lastTrack WikiTrackData
	var currentTrackNumber int
	var notes string

	for i, field := range matches {
		if i == 0 {
			// The first one is just itself
			continue
		}

		field := string(bytes.TrimSpace(field))

		switch i {
		case 1:
			if field != "" {
				parsedData.Lineage = "*" + field + "\n"
			}
		case 2:
			lineage := ""
			for _, item := range strings.Split(field, "\n") {
				lineage += "*" + strings.TrimSpace(item) + "\r\n"
			}
			parsedData.Lineage += lineage
		case 3:
			notes = field
		case 4:
			str, err := url.QueryUnescape(field)
			if err != nil {
				fmt.Println("error unescaping query from url")
				fmt.Println(err.Error())
				return
			}

			str = strings.Replace(str, "_", " ", -1) // make spaces in wikiformat real spaces
			str = strings.Replace(str, "/", "_", -1) // make slashes fileurl compliant by making it a "_"
			str = strings.Replace(str, ":", "💩", -1) // makes colons fileurl compliant by making it a pile of poo

			str = strings.Trim(strconv.QuoteToASCII(str), "\"") // make it ascii escaped, and trim "s
			str = strings.Replace(str, "\\", "^", -1)           // escape "\" with "^" so that the bash script can make it a codepoint again

			wikifile = fpath.Join(wikifile, str+".wiki")
			fmt.Printf("\n - %s... ", wikifile)
		case 5:
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

				currentTrackNumber++
				trackData.Index = currentTrackNumber

				lastTrack = trackData
				tracks = append(tracks, trackData)
			}
		case 6:
			parsedData.Duration = field
		}
	}
	parsedData.Tracks = tracks
	parsedData.Notes = bracketRegex.ReplaceAllStringFunc(notes, wikiReplace(tracks))

	success := util.RemoveFile(wikifile, false)
	if deleteMode {
		message := "success!"
		if !success {
			message = "couldn't delete!"
		}
		fmt.Println(message)
		return
	} else if success {
		fmt.Print("overwritten... ")
	}

	wikiout := util.CreateFile(wikifile)
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

func wikiReplace(tracks []WikiTrackData) func(string) string {
	return func(str string) string {
		trackName := str[1 : len(str)-1]
		for _, track := range tracks {
			if track.Name == trackName {
				return "[[" + trackName + "]]"
			}
		}
		return str
	}
}
