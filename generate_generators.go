package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"io/ioutil"
	"path"
	"strings"
	"text/template"
	"time"
)

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
		removeFile(outputFilename, true)
		return
	}

	album := new(AlbumData)
	album.Tour = tour.Name

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
				} else if (path.Ext(filename) == ".flac") && !isDir {
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

	albumDuration := time.Duration(0) // duration incrementer for the album
	for _, file := range iterating {
		track := getTagsFromFile(path.Join(filepath, file), album, &albumDuration)

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

	album.Duration = formatDuration(albumDuration)

	funcMap := template.FuncMap{"wikiescape": wikiescape}
	t := template.Must(template.New("generate").Funcs(funcMap).Parse(informationTemplate))

	fmt.Println("Creating", outputFilename+"...")
	infoFile := createFile(outputFilename)
	defer infoFile.Close()
	err := t.Execute(infoFile, album)
	if err != nil {
		panic(err)
	}
}
