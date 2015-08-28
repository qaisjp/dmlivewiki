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

	fmt.Printf("The following filepath will be processed: %s\n", filepath)

	if !shouldContinue(c) {
		return
	}

	// tour := c.Args()[0]

	fmt.Println("Trying to generate with")
}
