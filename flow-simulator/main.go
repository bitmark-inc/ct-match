package main

import (
	"log"
	"os"

	"github.com/bitmark-inc/pfizer/flow-simulator/config"
	"github.com/bitmark-inc/pfizer/flow-simulator/simulator"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	configFile string
)

func main() {
	app := cli.NewApp()
	app.Name = "simulator"
	app.Usage = "to simulate the flow for matching service"
	app.Action = func(c *cli.Context) error {
		conf, err := config.Load(configFile)
		if err != nil {
			return err
		}
		s := simulator.New(conf)
		return s.Simulate()
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "config, c",
			Value:       "",
			Usage:       "configuration file",
			Destination: &configFile,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
