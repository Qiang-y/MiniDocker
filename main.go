package main

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"os"
)

const usage = `Usage`

func main() {
	app := cli.NewApp()
	app.Name = "miniDocker"
	app.Usage = usage
	app.Commands = []*cli.Command{
		&runCommand,
		&initCommand,
		&commitCommand,
	}
	app.Before = func(ctx *cli.Context) error {
		logrus.SetFormatter(&logrus.JSONFormatter{})
		logrus.SetOutput(os.Stdout)
		return nil
	}
	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}
