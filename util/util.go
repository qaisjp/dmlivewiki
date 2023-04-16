package util

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	fpath "path/filepath"
	"strings"
	"time"

	"gopkg.in/urfave/cli.v1"
)

func WikiEscape(s string) string {
	return url.QueryEscape(strings.Replace(s, " ", "_", -1))
}

// not needed anymore
func CreateFile(filename string) *os.File {
	f, err := os.Create(filename)
	if err != nil {
		if os.IsPermission(err) {
			fmt.Println("permission error!")
			return nil
		}

		panic(err.Error())
	}
	return f
}

// a bit of a mess
func RemoveFile(filename string, log bool) bool {
	if log {
		fmt.Printf("Removing %s...", filename)
	}

	err := os.Remove(filename)
	if err != nil {
		if os.IsNotExist(err) {
			if log {
				fmt.Println(" does not exist!")
			}
			return false
		} else if os.IsPermission(err) {
			if log {
				fmt.Println(" permission error!")
			}
			return false
		}
		fmt.Println("Something happened when deleting your file! :(")
		fmt.Println(err.Error())
		return false
	}
	if log {
		fmt.Println(" success!")
	}
	return true
}

func isDirectory(filepath string) (bool, os.FileInfo, error) {
	fileInfo, err := os.Stat(filepath)
	if err != nil {
		return false, nil, err
	}
	return fileInfo.IsDir(), fileInfo, err
}

func GetDirectorySite(filepath string) (float64, error) {
	var size float64
	contents, err := ioutil.ReadDir(filepath)
	if err != nil {
		return -1, err
	}

	for _, file := range contents {
		if file.IsDir() {
			subsize, err := GetDirectorySite(fpath.Join(filepath, file.Name()))
			if err != nil {
				return -1, err
			}
			size += subsize
		} else {
			size += float64(file.Size())
		}
	}

	return size, nil
}

func GetFileErrorReason(err error) string {
	if os.IsNotExist(err) {
		return "doesn't exist"
	} else if os.IsPermission(err) {
		return "permission error"
	} else {
		return err.Error()
	}
}

func CheckFilepathArgument(c *cli.Context) (os.FileInfo, string) {
	if len(c.Args()) != 1 {
		if err := cli.ShowSubcommandHelp(c); err != nil {
			fmt.Println("Error:", err.Error())
		}
		return nil, ""
	}

	filepath := c.Args()[0]
	return GetFileOfType(filepath, true, "target")
}

func GetFileOfType(filepath string, wantDirectory bool, target string) (os.FileInfo, string) {
	filepath, err := fpath.Abs(filepath)
	if err != nil {
		fmt.Println("Could not find absolute directory for path. Error:")
		fmt.Println(err.Error())
		return nil, ""
	}

	isDirectory, fileInfo, _ := isDirectory(filepath)
	if (fileInfo == nil) || (isDirectory != wantDirectory) {
		if wantDirectory {
			fmt.Println("Error:", target, "is not a directory")
		} else {
			fmt.Println("Error:", target, "is not a file")
		}
		return nil, ""
	}

	return fileInfo, filepath
}

func ShouldContinue(c *cli.Context) bool {
	// Ask to continue or just process?
	if c.GlobalBool("force") {
		fmt.Print("\n")
		return true
	}

	fmt.Print("Continue? (y/n): ")
	text := ""
	fmt.Scanln(&text)
	fmt.Print("\n")

	return text == "y" || text == "Y"
}

func NotifyDeleteMode(c *cli.Context) {
	if c.GlobalBool("delete") {
		fmt.Println("You are running in **DELETE MODE** - data will be permanently lost")
	}
}

// there is a better way to do this
func FormatDuration(d time.Duration) (str string) {
	hours := int(d.Hours())
	d -= (time.Duration(hours) * time.Hour)

	minutes := int(d.Minutes())
	d -= (time.Duration(minutes) * time.Minute)

	seconds := int(d.Seconds())

	if hours == 0 {
		str += fmt.Sprintf("%d:%02d", minutes, seconds)
	} else {
		str += fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return
}
