package main

import (
	"bufio"
	"errors"
	"os"
	"strings"
)

type Tour struct {
	Name   string
	Tracks map[string]struct{}
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
