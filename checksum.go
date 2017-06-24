package main

import (
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

func performChecksum(c *cli.Context) {
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

	workingDirectory, err := os.Getwd()
	if err != nil {
		fmt.Println("could not get working directory for some reason")
		fmt.Println("reason is: " + err.Error())
		fmt.Println("aborting!")
		return
	}

	if mode == "single" {
		checksumProcessPath(filepath, fileInfo.Name(), c.GlobalBool("delete"), workingDirectory)
		return
	}

	files, _ := ioutil.ReadDir(filepath)
	for _, file := range files {
		if file.IsDir() {
			checksumProcessPath(fpath.Join(filepath, file.Name()), file.Name(), c.GlobalBool("delete"), workingDirectory)
		}
	}
}

func checksumProcessPath(directory string, name string, deleteMode bool, workingDirectory string) {
	directory = fpath.Clean(directory)
	baseFilename := fpath.Join(directory, name)
	ffpFilename := baseFilename + ".ffp"
	md5Filename := baseFilename + ".md5"

	// If we're in delete mode, let's just delete the ffp and md5 files right away
	if deleteMode {
		util.RemoveFile(ffpFilename, true)
		util.RemoveFile(md5Filename, true)
		return
	}

	// Let's create an md5 file buffer and
	// a pool to store files to be in the ffp
	var md5Buffer bytes.Buffer
	ffpPool := []string{
		// The first arg for ffpPool is this
		// because we're going to dump the entire
		// pool in the command later
		"--show-md5sum",
		"--no-filename",
	}

	// This walks through every file in the folder
	fpath.Walk(directory,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Println("!!Encountered error for: " + path)
				fmt.Println("!!This is the message: " + err.Error())
				return nil
			} else if info.IsDir() {
				// We don't care about directories either,
				// so let's jump out of here
				return nil
			}

			if (path == ffpFilename) || (path == md5Filename) {
				// ffpFilename is hashed afterwards
				// and we don't need to hash ourself either
				return nil
			}

			name := strings.TrimPrefix(
				path,
				directory+string(os.PathSeparator),
			)

			if fpath.Ext(path) == ".flac" {
				// So if the file we have is an ffp file,
				// lets add it to the pool to be checked!
				ffpPool = append(ffpPool, name)
			}

			// Read the file
			data, err := ioutil.ReadFile(path)

			if err != nil {
				fmt.Println("!!Encountered error for: " + path)
				fmt.Println("!!This is the message: " + err.Error())
			}

			md5Buffer.WriteString(checksumFormatMD5(md5.Sum(data), strings.TrimPrefix(name, directory+"/")))
			return nil
		},
	)

	// If the pool contains atleast one **filename**
	// the first two items in the pool is actually just a flag!
	if len(ffpPool) > 2 {
		cmd := exec.Command("metaflac", ffpPool[:]...)
		cmd.Dir = directory

		data, err := cmd.Output()
		if err != nil {
			fmt.Println("metaflac returned an invalid response")
			if data != nil {
				fmt.Println(data)
			}
			panic(err)
		}

		hashes := strings.Split(string(data), "\n")
		for i, hash := range hashes {
			if (i == len(hashes)-1) && (hash == "") {
				// Last line is an empty line
				continue
			}

			hashes[i] = strings.TrimSpace(fmt.Sprintf("%s:%s", strings.TrimPrefix(ffpPool[i+2], directory), hash))
		}
		data = []byte(strings.Join(hashes, "\r\n"))

		// The md5 buffer doesn't contain our ffp file, so let's write that to the buffer
		md5Buffer.WriteString(checksumFormatMD5(md5.Sum(data), name+".ffp"))

		// Let's write the ffp file now
		ffp, err := os.Create(ffpFilename)
		defer ffp.Close()

		if err != nil {
			fmt.Println("!!Could not create ffp file: " + ffpFilename)
			fmt.Println("!!Error: " + err.Error())
		} else {
			ffp.Write(data)
		}
	}

	// If the md5buffer isn't empty
	if md5Buffer.Len() > 0 {
		// Let's write the md5 file
		md5, err := os.Create(md5Filename)
		defer md5.Close()

		if err != nil {
			fmt.Println("!!Could not create md5 file: " + md5Filename)
			fmt.Println("!!Error: " + err.Error())
		} else {
			md5.Write(md5Buffer.Bytes())
		}
	}

	fmt.Println("Done with", directory)
}

func checksumFormatMD5(hash [16]byte, name string) string {
	return fmt.Sprintf("%x *%s\r\n", hash, name)
}
