package main

import (
	"fmt"
	"github.com/codegangsta/cli"
)

func generateInformation(c *cli.Context) {
	filepath, _ := checkFilepathArgument(c)
	if filepath == "" {
		return
	}

	if !shouldContinue(c, filepath) {
		return
	}

	// tour := c.Args()[0]

	fmt.Println("Trying to generate with")
}
