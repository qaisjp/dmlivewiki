package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"gopkg.in/urfave/cli.v1"
)

var metaflacPath string

func findMetaflac() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// _, err = os.Stat("./metaflac")
	// if err == nil {
	// 	// absolute path to metaflac relative to cwd
	// 	return filepath.Join(cwd, "metaflac"), nil
	// }

	path, err := exec.LookPath("metaflac")
	if errors.Is(err, exec.ErrDot) {
		return cwd + "/" + path, nil
	} else if err != nil {
		return "", err
	}
	return path, nil
}

func main() {
	var err error
	metaflacPath, err = findMetaflac()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	configPath := os.Getenv("config")
	if configPath == "" {
		configPath = "config.yaml"
	}
	if len(os.Args) > 1 {
		if err := parseConfig(configPath); err != nil {
			fmt.Println(err.Error())
			return
		}
	}

	app := cli.NewApp()
	app.Name = "dmlivewiki"
	app.Usage = "dmlivewiki helper"
	app.Author = `Qais "qaisjp" Patankar`
	app.Email = "me@qaisjp.com"
	app.Version = "1.1"

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "force, f",
			Usage: "skip confirmation",
		},
		cli.BoolFlag{
			Name:  "delete",
			Usage: "instead of creating files, delete files",
		},
		cli.BoolFlag{
			Name:  "single, s",
			Usage: "parse the directory given, not the subdirectories",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:   "checksum",
			Usage:  "perform a checksum of directories",
			Action: performChecksum,
		},
		{
			Name:   "verify",
			Usage:  "verify ffp and md5 files in directories",
			Action: verifyChecksum,
		},
		{
			Name:   "generate",
			Usage:  "generate dirname.txt Infofile's for the passed directory",
			Action: generateInformation,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "tour",
					Usage: "required: the tour name for this directory",
				},
				cli.StringFlag{
					Name:  "tour-file",
					Usage: "file with list of tracks with alternate vocals",
				},
			},
		},
		{
			Name:   "wiki",
			Usage:  "generate dirname.wiki Wikifile's for the passed directory",
			Action: generateWikifiles,
		},
		{
			Name:   "find",
			Usage:  "finds unfilled .txt files for the passed directory",
			Action: findWikifiles,
		},
	}

	app.Run(os.Args)
}

// Information template to write .txt info files from a folder
var informationTemplate = `{{.Artist}}
{{.Date}}
{{.Album}}
{{.Tour}}

Lineage: 

Notes: 

This source is considered Source 1 for this date:
$$wikiPath$$/{{wikiescape .Date}}_{{wikiescape .Album}}/Source_1

Track list:

{{range .Tracks}}{{.Prefix}}{{printf "%02d" .Index}}. [{{.Duration}}] {{.Title}}{{if .HasAlternateLeadVocalist}} (*){{end}}
{{end}}Total time: {{.Duration}}

$$footer$$`

// Wiki template to write the .wiki files from edited .txt info files
var wikiTemplate = `== Notes ==

{{.Notes}}

== Listen ==

You can listen to this entire recording below.

<html5media>$$streamPath$$/{{.FolderName}}/complete.m4a</html5media>

== Track list ==

{{range .Tracks}}{{.LinePrefix}}[{{.Duration}}] <sm2>$$streamPath$$/{{.FolderName}}/{{printf "%02d" .Index}}.m4a</sm2> [[{{.Name}}]]{{if .HasAlternateLeadVocalist}} {{"{{"}}tt|(*)|Vocals by Martin Gore{{"}}"}}{{end}}
{{end}}*Total time: {{.Duration}}

== Lineage ==

{{.Lineage}}
== Download ==

*[$$downloadPath$$/{{.FolderName}}.zip Download ZIP] - FLAC {{.BPS}}-bit {{.SampleRate}} - {{.Size}}

[[Category:Audience recordings]]
[[Category:Source]]
[[Category:Streamable]]
`

// Wiki regex to read an edited .txt info file and extract what is needed for a .wiki file
const wikiRegexText = `((?:.*[\r\n])?)Lineage: ((?:.|[\r\n]+)*)[\r\n]+Notes: ((?:.|[\r\n]+)*)[\r\n]+This source is conside(?:.|[\r\n])*wiki\/(.*)[\r\n]*Track list:[\r\n]+[\r\n]+((?:.|[\r\n]+)*)[\r\n]+Total time: (.*)`
