package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"fmt"
	"github.com/codegangsta/cli"
	"io/ioutil"
	"os"
	"os/exec"
	fpath "path/filepath"
	// "strings"
)

var tick, cross string = "✔", "✖"

func verifyChecksum(c *cli.Context) {
	fileInfo, filepath := checkFilepathArgument(c)
	if fileInfo == nil {
		return
	}

	if c.GlobalBool("delete") {
		fmt.Println(`"delete" doesn't apply to this commmand`)
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

	// We need to get the working directory
	// so that we can get the path to metaflac
	workingDirectory, err := os.Getwd()
	if err != nil {
		fmt.Println("could not get working directory for some reason")
		fmt.Println("reason is: " + err.Error())
		fmt.Println("aborting!")
		return
	}

	if mode == "single" {
		verifyProcessPath(filepath, fileInfo.Name(), workingDirectory)
		return
	}

	files, _ := ioutil.ReadDir(filepath)
	for _, file := range files {
		if file.IsDir() {
			verifyProcessPath(fpath.Join(filepath, file.Name()), file.Name(), workingDirectory)
		}
	}
}

func verifyProcessPath(directory string, name string, workingDirectory string) {
	// Let us know what is currently being processed
	fmt.Print(directory + "... ")

	baseFilename := fpath.Join(directory, name+".")
	ffpFilename := baseFilename + "ffp"
	md5Filename := baseFilename + "md5"

	_, ffpErr := os.Stat(ffpFilename)
	_, md5Err := os.Stat(md5Filename)
	for _, err := range []error{ffpErr, md5Err} {
		if err != nil {
			file := "ffp"
			if err == md5Err {
				file = "md5"
			}
			fmt.Printf("\n> error with %s (%s)", file, getFileErrorReason(err))
		}
	}

	md5Success, md5ReadError := false, false
	if md5Err == nil {
		md5Success, md5ReadError = verifyMD5(md5Filename, directory)
	}

	ffpSuccess := false
	if md5ReadError {
		fmt.Printf("\n> skipping ffp check because of md5 file errors")
	} else if ffpErr == nil {
		ffpSuccess = verifyFFP(ffpFilename, directory, workingDirectory)
	}

	if md5Success && ffpSuccess {
		fmt.Println(tick)
		return
	} else {
		md5out, ffpOut := cross, cross
		if md5Success {
			md5out = tick
		}
		if ffpSuccess {
			ffpOut = tick
		}

		fmt.Printf("\n> done! ffp(%s) md5(%s)\n\n", ffpOut, md5out)
	}
}

// verify an md5 file against a directory
func verifyMD5(md5Filename string, directory string) (success, readError bool) {
	file, err := os.Open(md5Filename)
	defer file.Close()
	if err != nil {
		fmt.Printf("\n> md5 read err (%s)", err.Error())

		// we won't return readError at true because that's intended
		// for individual file read errors! this should be clearer
		return
	}

	reader := bufio.NewReader(file)
	scanner := bufio.NewScanner(reader)
	success = true

	for scanner.Scan() {
		line := scanner.Text()

		checksum := line[:32]
		filename := line[34:]

		// Read the file
		data, err := ioutil.ReadFile(fpath.Join(directory, filename))
		if err != nil {
			fmt.Printf("\n> md5 read error with %s (%s)", filename, getFileErrorReason(err))
			success = false
			readError = true
		}

		if fmt.Sprintf("%x", md5.Sum(data)) != checksum {
			fmt.Printf("\n> md5 mismatch for \"%s\"", filename)
			success = false
		}
	}
	return
}

// verify an ffp file against a directory
func verifyFFP(ffpFilename string, directory string, workingDirectory string) bool {
	file, err := os.Open(ffpFilename)
	defer file.Close()
	if err != nil {
		fmt.Printf("\n> ffp read err (%s)", err.Error())
		return false
	}

	reader := bufio.NewReader(file)
	scanner := bufio.NewScanner(reader)

	// we use this to store our actual ffp contents
	var ffpFileBuffer bytes.Buffer

	// for the metaflac execution
	files := []string{
		// The first arg for ffpPool is this
		// because we're going to dump the entire
		// pool in the command later
		"--show-md5sum",
	}

	for scanner.Scan() {
		line := scanner.Text()
		ffpFileBuffer.WriteString(line)
		ffpFileBuffer.WriteString("\r\n") // equality check won't be nice x-platform

		// The line has to be atleast 34 characters long
		if len(line) < 34 {
			fmt.Print("\n> ffp has an incorrect format")
			return false
		}

		md5sumIndex := len(line) - 32
		// checksum := line[md5sumIndex:]
		filename := line[:md5sumIndex-1]
		files = append(files, filename)

		// we don't need to check if it exists, because
		// md5 told our function caller not to call us
		// if mr. md5 couldn't find us
	}

	cmd := exec.Command(fpath.Join(workingDirectory, "metaflac"), files[:]...)
	cmd.Dir = directory

	data, err := cmd.Output()
	if err != nil {
		fmt.Println("\n> ffp metaflac error: ")
		if data != nil {
			fmt.Print(">> ", data)
		}
		return false
	}

	return bytes.Equal(data, ffpFileBuffer.Bytes())
}
