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
	}

	app.Commands = []cli.Command{
		{
			Name:   "checksum",
			Usage:  "perform a checksum of directories",
			Action: performChecksum,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "single, s",
					Usage: "parse the directory given, not the subdirectories",
				},
			},
		},
	}

	app.Run(os.Args)
}
