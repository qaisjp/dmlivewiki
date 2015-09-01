package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"io/ioutil"
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

{{range .Tracks}}{{.Prefix}}{{printf "%02d" .Index}} [{{.Duration}}] {{.Title}}{{if .HasAlternateLeadVocalist}} (*){{end}}
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
	Title                    string
	Duration                 string
	HasAlternateLeadVocalist bool
	Prefix                   string
	Index                    int
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
		fileInfo, tourfileClean := getFileOfType(tourfile, false, "tour-file")
		if fileInfo == nil {
			return
		}
		tourfile = tourfileClean
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
	outputFilename := path.Join(filepath, name+".txt")
	if deleteMode {
		removeFile(outputFilename)
		return
	}

	album := new(AlbumData)
	album.Tour = tour.Name

	var duration int64 = 0 // duration incrementer for the album

	useCDNames := false
	folders := make([]string, 0)
	extraFolders := make([]string, 0)
	files := make([]string, 0)

	directoryContents, _ := ioutil.ReadDir(filepath)
	for _, fileinfo := range directoryContents {
		filename := fileinfo.Name()
		isDir := fileinfo.IsDir()
		if isDir {
			if strings.HasPrefix(filename, "CD") {
				folders = append(folders, filename)
				useCDNames = true
			} else {
				extraFolders = append(extraFolders, filename)
			}
		} else if (path.Ext(filename) == ".flac") && !isDir {
			files = append(files, filename)
		}
	}

	iterating := files
	if useCDNames {

		if len(files) > 0 {
			// Contains extra files not in a specific CD
			// Do something!
			fmt.Println("Warning! Files outside CD folders in", filepath)
		}

		files := make([]string, 0)
		subfolders := make([]string, 0)
		for _, dirName := range folders {
			subdirectory, _ := ioutil.ReadDir(path.Join(filepath, dirName))
			for _, fileinfo := range subdirectory {
				subdirPath := path.Join(dirName, fileinfo.Name())
				if fileinfo.IsDir() {
					subfolders = append(subfolders, subdirPath)
				} else {
					files = append(files, subdirPath)
				}
			}
		}

		if len(subfolders) > 0 {
			fmt.Printf("Skipping! Filepath has depth=3 folders (%s)\n", filepath)
			return
		}

		iterating = files // set it to the new files
		// this means old files won't be iterated
	}

	if len(extraFolders) > 0 {
		// Contains extra folders, do something!
		// There's probably a folder like "Bonus"
		fmt.Println("Warning! Extra non CD folders inside", filepath)
	}

	for _, file := range iterating {
		track := getTagsFromFile(path.Join(filepath, file), album, &duration)

		if tour.Tracks != nil {
			_, containsAlternateLeadVocalist := tour.Tracks[track.Title]
			track.HasAlternateLeadVocalist = containsAlternateLeadVocalist
		}

		if useCDNames {
			track.Prefix = strings.TrimPrefix(path.Dir(file), "CD") + "."
		}

		// Finally, add the new track to the album
		album.Tracks = append(album.Tracks, track)
	}

	if len(album.Tracks) == 0 {
		fmt.Println("Could not create album - aborting creation of", outputFilename)
		return
	}

	format := "4:05" // minute:0second
	if duration >= 3600 {
		format = "15:04:05" // duration is longer than an hour
	}
	album.Duration = time.Unix(duration, 0).Format(format)

	funcMap := template.FuncMap{"wikiescape": wikiescape}
	t := template.Must(template.New("generate").Funcs(funcMap).Parse(informationTemplate))

	infoFile := createFile(outputFilename)
	defer infoFile.Close()
	err := t.Execute(infoFile, album)
	if err != nil {
		panic(err)
	}
}

// tags: http://age.hobba.nl/audio/tag_frame_reference.html
func getTagsFromFile(filepath string, album *AlbumData, albumDuration *int64) TrackData {
	args := []string{
		"--show-total-samples",
		"--show-sample-rate",
	}

	nonTagArgs := len(args)
	tags := []string{"TITLE", "tracknumber"}

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
			case "tracknumber":
				num, err := strconv.Atoi(tagValue)
				if err != nil {
					panic(err)
				}

				track.Index = num
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

	return track
}
