package main

import (
	"github.com/codegangsta/cli"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Name = "dmlivewiki"
	app.Usage = "dmlivewiki helper"
	app.Author = `Qais "qaisjp" Patankar`
	app.Email = "me@qaisjp.com"
	app.Version = "1.0.6"

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
	}

	app.Run(os.Args)
}

var informationTemplate = `{{.Artist}}
{{.Date}}
{{.Album}}
{{.Tour}}

Lineage: 

Notes: 

This source is considered Source 1 for this date:
https://www.depechemode-live.com/wiki/{{wikiescape .Date}}_{{wikiescape .Album}}/Source_1

Track list:

{{range .Tracks}}{{.Prefix}}{{printf "%02d" .Index}}. [{{.Duration}}] {{.Title}}{{if .HasAlternateLeadVocalist}} (*){{end}}
{{end}}Total time: {{.Duration}}

Torrent downloaded from https://www.depechemode-live.com`

var wikiTemplate = `== Notes ==

{{.Notes}}

== Listen ==

You can listen to this entire recording below.

<html5media>https://media.depechemode-live.com/stream/{{.FolderName}}/complete.m4a</html5media>

== Track list ==

{{range .Tracks}}#[{{.Duration}}] <sm2>https://media.depechemode-live.com/stream/{{.FolderName}}/{{printf "%02d" .Index}}.m4a</sm2> [[{{.Name}}]]{{if .HasAlternateLeadVocalist}} (*){{end}}
{{end}}*Total time: {{.Duration}}

== Lineage ==

{{.Lineage}}

== Download ==

*[https://depechemode-live.com/torrents/{{.FolderName}}.torrent Download via torrent] - FLAC {{.BPS}}-bit {{.SampleRate}} - {{.Size}}

[[Category:Audience recordings]]
[[Category:Source]]
[[Category:Streamable]]
`

var wikiRegex = `(?:.|[\r\n])+[\r\n]+Lineage: ((?:.|[\r\n]+)*)[\r\n]+Notes: ((?:.|[\r\n]+)*)[\r\n]+This source is considered(?:.|[\r\n]+)*Track list:[\r\n]+[\r\n]+((?:.|[\r\n]+)*)[\r\n]+Total time: (.*)`
