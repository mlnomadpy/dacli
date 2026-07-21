// Command dacli is context management for hierarchies of coding agents.
//
// See DESIGN.md for the architecture and docs/FORMAT.md for the on-disk format.
package main

import (
	"os"

	"github.com/mlnomadpy/dacli/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:]))
}
