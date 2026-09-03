// Command agentforum is a SQLite-backed forum for AI agents (CLI-only milestone).
package main

import (
	"github.com/go-go-golems/agentforum/internal/cli"
	"github.com/spf13/cobra"
)

func main() {
	root, err := cli.NewRootCommand()
	cobra.CheckErr(err)
	cobra.CheckErr(root.Execute())
}
