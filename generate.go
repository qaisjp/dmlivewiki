package main

import (
	"fmt"
	"github.com/codegangsta/cli"
)

func generateInformation(c *cli.Context) {
	if !checkCommandArgumentNumber(c, 1) {
		return
	}

	fmt.Println("Trying to generate with")
}
