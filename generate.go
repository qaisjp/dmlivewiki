package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"text/template"
	"time"
)

var informationTemplate = `{{.Artist}}
{{.Date}}
{{.Album}}
{{.Tour}}

Lineage: 

Notes: 

This source is considered Source 1 for this date:
https://www.depechemode-live.com/wiki/{{wikiescape .Date}}_{{wikiescape .Album}}/Source_1

Track list:

{{range .Tracks}}{{printf "%02d" .Index}}. [{{.Duration}}] {{.Title}}{{if .HasAlternateLeadVocalist}} (*){{end}}
{{end}}Total time: {{.Duration}}

Torrent downloaded from https://www.depechemode-live.com
`

type AlbumData struct {
	Artist   string
	Date     string
	Album    string
	Tour     string
	Tracks   []TrackData
	Duration string
}

type TrackData struct {
	Index                    int
	Title                    string
	Duration                 string
	HasAlternateLeadVocalist bool
}

func generateInformation(c *cli.Context) {
	fileInfo, filepath := checkFilepathArgument(c)
	if fileInfo == nil {
		return
	}

	tourName := c.String("tour")
	if tourName == "" {
		cli.ShowSubcommandHelp(c)
		return
	}

	mode := "batch"
	if c.GlobalBool("single") {
		mode = "single"
	}

	tourfile := c.String("tour-file")
	if tourfile != "" {
		fileInfo, tourfile = getFileOfType(tourfile, false, "tour-file")
		if fileInfo == nil {
			return
		}

		fmt.Println("Processing tours from:", tourfile)
	}

	fmt.Println("The current tour is:", tourName)
	fmt.Printf("The following filepath (%s mode) will be processed: %s\n", mode, filepath)
	notifyDeleteMode(c)

	if !shouldContinue(c) {
		return
	}

	tour := new(Tour)
	tour.Name = tourName
	if tourfile != "" { // tourFile is only for reading "alternate vocalists" into tracks map
		if err := getTourFromTourFile(tourfile, tour); err != nil {
			fmt.Println("[Error]", err)
			if !shouldContinue(c) {
				return
			}
		}
	}

	// Stupid windows
	informationTemplate = strings.Replace(informationTemplate, "\n", "\r\n", -1)

	if mode == "single" {
		generateFile(filepath, fileInfo.Name(), *tour, c.GlobalBool("delete"))
		return
	}

	files, _ := ioutil.ReadDir(filepath)
	for _, file := range files {
		if file.IsDir() {
			name := file.Name()
			generateFile(path.Join(filepath, name), name, *tour, c.GlobalBool("delete"))
		}
	}
}

func generateFile(filepath string, name string, tour Tour, deleteMode bool) {
	var infoFile *os.File
	if filename := path.Join(filepath, name+".txt"); deleteMode {
		removeFile(filename)
		return
	} else {
		infoFile = createFile(filename)
		defer infoFile.Close()
	}

	album := new(AlbumData)
	album.Tour = tour.Name

	var duration int64 = 0 // duration incrementer for the album

	files, _ := ioutil.ReadDir(filepath)
	for _, file := range files {
		if name := file.Name(); (path.Ext(name) == ".flac") && !file.IsDir() {
			index := getTagsFromFile(path.Join(filepath, name), album, &duration)

			if tour.Tracks != nil {
				_, containsAlternateLeadVocalist := tour.Tracks[album.Tracks[index].Title]
				album.Tracks[index].HasAlternateLeadVocalist = containsAlternateLeadVocalist
			}
		}
	}

	format := "4:05" // minute:0second
	if duration >= 3600 {
		format = "15:04:05" // duration is longer than an hour
	}
	album.Duration = time.Unix(duration, 0).Format(format)

	funcMap := template.FuncMap{"wikiescape": wikiescape}
	t := template.Must(template.New("generate").Funcs(funcMap).Parse(informationTemplate))
	err := t.Execute(infoFile, album)
	if err != nil {
		panic(err)
	}
}

func getTagsFromFile(filepath string, album *AlbumData, albumDuration *int64) int {
	args := []string{
		"--show-total-samples",
		"--show-sample-rate",
	}

	nonTagArgs := len(args)
	tags := []string{"TITLE"}

	getAlbumData := album.Artist == ""
	if getAlbumData {
		tags = append(tags,
			"ARTIST",
			"DATE",
			"ALBUM",
		)
	}

	args = append(args, filepath)
	for _, tag := range tags {
		args = append(args, "--show-tag="+tag)
	}

	data, err := exec.Command(
		"metaflac",
		args[:]...,
	).Output()

	if err != nil {
		panic(err)
	}

	var track TrackData
	track.Index = len(album.Tracks) + 1

	lines := strings.Split(string(data), "\r\n")
	if len(lines) != len(args) {
		panic(fmt.Sprintf("[invalid metaflac output] Expected %d lines, got %d", len(args), len(lines)-1))
		// todo, return a bool to delete this file
		// and say that the current file is being skipped
		// perhaps an --ignore flag to enable this feature
		// false by default, to make it cancel the whole procedure?
	}

	var samples, sampleRate int64
	for i, line := range lines {
		line = strings.TrimSpace(line)

		switch {
		case i <= 1:
			value, err := strconv.Atoi(line)
			if err != nil {
				panic(err)
			}

			if i == 0 {
				samples = int64(value)
			} else {
				sampleRate = int64(value)
			}
		case i < len(args)-1:
			tagName := tags[i-nonTagArgs]
			prefix := tagName + "="
			tagValue := ifTrimPrefix(line, prefix)

			switch tagName {
			case "TITLE":
				track.Title = tagValue
			case "ARTIST":
				album.Artist = tagValue
			case "DATE":
				album.Date = tagValue
			case "ALBUM":
				album.Album = ifTrimPrefix(tagValue, album.Date+" ")
			}
		}
	}
	duration := samples / sampleRate
	*albumDuration += duration
	track.Duration = time.Unix(duration, 0).Format("4:05")

	// Finally, add the new track to the album
	album.Tracks = append(album.Tracks, track)
	return len(album.Tracks) - 1
}
