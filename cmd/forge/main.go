// Command forge exercises the forge package from the CLI without IGDB credentials.
package main

import (
	"log"
	"os"

	"vargames-name-gen/src/cli"
)

func main() {
	if err := cli.RunForge(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
