package cmd

import (
	"os"

	"github.com/urfave/cli"
)

// Execute
func Execute() {
	app := cli.NewApp()
	app.Name = "graphql-orm"
	app.Usage = "This tool is for generating "
	app.Version = "0.0.1"

	app.Commands = []cli.Command{lockCmd, unlockCmd}

	app.Run(os.Args)
}
