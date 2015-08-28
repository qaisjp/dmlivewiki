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
	app.Version = "1.0.3"

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
			Usage:  "generate info.txt file for the passed directory",
			Action: generateInformation,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "tour",
					Usage: "required: the tour name for this directory",
				},
			},
		},
	}

	app.Run(os.Args)
}
