package main

import (
	"context"
	"os"
	"runtime/debug"
	"sync"

	"github.com/pydpll/errorutils"
	"github.com/urfave/cli/v3"
	"go.etcd.io/bbolt"
)

var (
	CommitID    = CommitIDer()
	worker      workers
	shouldCount bool
	altPrint    bool
)

type workers func(db *bbolt.DB, KEYS_ch <-chan string, RETURN_ch chan<- string, wg *sync.WaitGroup)

func main() {
	err := app.Run(context.Background(), os.Args)
	errorutils.ExitOnFail(err)
}

var app = cli.Command{
	Name:      "monkey",
	Usage:     "branch swing on bbolt trees and give key@value formatted bananas to store",
	ArgsUsage: "<file.bbolt> [file2.bbolt ... ]",
	Version:   CommitID,
	Action:    readFiles,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:        "count",
			Usage:       "change tree view to not show leaves",
			Aliases:     []string{"c"},
			Destination: &shouldCount,
		},
		&cli.BoolFlag{
			Name:        "altPrint",
			Usage:       "change tree view to colon separated path",
			Aliases:     []string{"p"},
			Destination: &altPrint,
		},
	},
	Commands: []*cli.Command{
		{
			Name:               "grasp",
			Aliases:            []string{"hold"},
			Category:           "monkey business",
			Usage:              "save bananas with key@value format. Use single quotes in each token separately to handles spaces",
			Action:             grasp,
			ArgsUsage:          "<key1@value1> <key2@'value 2 with spaces needs single quotes'> ... [file.bbolt]",
			CustomHelpTemplate: monkeyBusinessHelp,
			Version:            CommitID,
		},
		{
			Name:               "deliver",
			Aliases:            []string{"show"},
			Category:           "monkey business",
			Usage:              "show bananas with separate key arguments. Exit code 0 if not found",
			Action:             deliver,
			ArgsUsage:          "<key1> <key2> ... [file.bbolt]",
			CustomHelpTemplate: monkeyBusinessHelp,
			Version:            CommitID,
		},
		{
			Name:               "hurl",
			Aliases:            []string{"throw"},
			Category:           "monkey business",
			Usage:              "delete key@value",
			Action:             hurl,
			ArgsUsage:          "<key1> <key2> ... [file.bbolt]",
			CustomHelpTemplate: monkeyBusinessHelp,
			Version:            CommitID,
		},
	},
}

func CommitIDer() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value[:12]
			}
		}
	}
	return ""
}

var monkeyBusinessHelp = `{{.Name}} - {{.Usage}}

	USAGE:  monkey {{.Name}} {{.ArgsUsage}}
	{{if len .Names}}ALIASES: {{join .Names ", "}}
	{{end}}{{if .Version}}VERSION: {{.Version}}
{{end}}`
