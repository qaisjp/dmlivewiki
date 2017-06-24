package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	fpath "path/filepath"
	"strings"

	"github.com/qaisjp/dmlivewiki/util"

	"gopkg.in/urfave/cli.v1"
)

var tick, cross string = `ok`, `bad`

func verifyChecksum(c *cli.Context) {
	fileInfo, filepath := util.CheckFilepathArgument(c)
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
	util.NotifyDeleteMode(c)

	if !util.ShouldContinue(c) {
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
		if file.IsDir() && file.Name() != "__wikifiles" {
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
			fmt.Printf("\n> %s read error: (%s)", file, util.GetFileErrorReason(err))
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
	}

	md5out, ffpOut := cross, cross
	if md5Success {
		md5out = tick
	}
	if ffpSuccess {
		ffpOut = tick
	}

	fmt.Printf("\n> done! ffp(%s) md5(%s)\n\n", ffpOut, md5out)
}

// verify an md5 file against a directory
func verifyMD5(md5Filename string, directory string) (success, readError bool) {
	file, err := os.Open(md5Filename)
	defer file.Close()
	if err != nil {
		fmt.Printf("\n> md5: read err (%s)", err.Error())

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
			fmt.Printf("\n> md5: read error with %s (%s)", filename, util.GetFileErrorReason(err))
			success = false
			readError = true
		}

		if fmt.Sprintf("%x", md5.Sum(data)) != checksum {
			fmt.Printf("\n> md5: mismatch for \"%s\"", filename)
			success = false
		}
	}
	return
}

// verify an ffp file against a directory
func verifyFFP(ffpFilename string, directory string, workingDirectory string) (success bool) {
	file, err := os.Open(ffpFilename)
	defer file.Close()
	if err != nil {
		fmt.Printf("\n> ffp: read err (%s)", err.Error())
		return
	}

	var ffpFileBuffer bytes.Buffer // contains actual ffp content
	var files, checksums []string

	reader := bufio.NewReader(file)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		ffpFileBuffer.WriteString(line + "\r\n")

		// The line has to be atleast 34 characters long
		if len(line) < 34 {
			fmt.Print("\n> ffp: incorrect line format")
			fmt.Printf("\n>> content (len:%d): %s", len(line), line)
			continue
		}

		filename, checksum := verifyReadFFP(line)

		_, err := os.Stat(fpath.Join(directory, filename))
		if err == nil {
			files = append(files, filename)
			checksums = append(checksums, checksum)
		} else {
			fmt.Printf("\n> ffp: \"%s\" has a problem (%s)", filename, util.GetFileErrorReason(err))
		}
	}

	if len(files) == 0 {
		fmt.Print("\n> ffp file contains no valid flac files")
		return
	}

	var cmdStdout, cmdStderr bytes.Buffer
	cmdArgs := append(
		[]string{
			"--show-md5sum",
		},
		files[:]...,
	)

	cmd := exec.Command("metaflac", cmdArgs[:]...)
	cmd.Dir = directory
	cmd.Stderr = &cmdStderr
	cmd.Stdout = &cmdStdout

	err = cmd.Run()
	if err != nil {
		fmt.Printf("\n> ffp metaflac error (%s)", err.Error())
	}

	if cmdStderr.Len() != 0 {
		fmt.Print("\n> ffp metaflac returned error info, dumping output: \n\t",
			// Replace every new line of the stderr with an indentation
			strings.TrimSpace(strings.Replace(
				cmdStderr.String(),
				"\r\n",
				"\n\t",
				-1,
			)),
		)

		if cmdStdout.Len() == 0 {
			return
		}
	}

	success = true

	// first do a quick equality check
	if bytes.Equal(cmdStdout.Bytes(), ffpFileBuffer.Bytes()) {
		// we don't need to check line by line
		return
	}

	// some things failed, so now we need to go through each one
	stdoutScanner := bufio.NewScanner(bufio.NewReader(&cmdStdout))
	for stdoutScanner.Scan() {
		line := stdoutScanner.Text()

		for i, file := range files {
			filename, checksum := verifyReadFFP(line)
			if filename == file {
				if checksum != checksums[i] {
					fmt.Printf("\n> ffp: mismatch for \"%s\"", filename)
					success = false
				}

				// remove this the verifying file from the files checking list
				files = append(files[:i], files[i+1:]...)
				checksums = append(checksums[:i], checksums[i+1:]...)
				break
			}

		}
	}

	if len(files) != 0 {
		fmt.Printf("\n> ffp: metaflac didn't like:\n\t%s", strings.Join(files, "\n\t"))
		success = false
	}

	return
}

func verifyReadFFP(line string) (filename, checksum string) {
	md5sumIndex := len(line) - 32
	return line[:md5sumIndex-1], line[md5sumIndex:]
}
