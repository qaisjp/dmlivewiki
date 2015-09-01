package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Tour struct {
	Name   string
	Tracks map[string]struct{}
}

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

func getTourFromTourFile(filepath string, tour *Tour) error {
	file, err := os.Open(filepath)
	defer file.Close()
	if err != nil {
		return errors.New("Could not open Tourfile (" + err.Error() + ")")
	}

	reader := bufio.NewReader(file)
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()
		prefix := tour.Name + ":"
		if !strings.HasPrefix(line, prefix) {
			continue
		}

		list := strings.TrimSpace(strings.TrimPrefix(line, prefix))
		tracks := strings.Split(list, ",")
		tour.Tracks = make(map[string]struct{})

		for _, track := range tracks {
			tour.Tracks[strings.TrimSpace(track)] = struct{}{}
		}
		return nil
	}
	return errors.New("Tourfile does not contain tour")
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
