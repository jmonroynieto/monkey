package main

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pydpll/errorutils"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
	"go.etcd.io/bbolt"
)

const (
	// readble bools 'keysOnly'
	parseLoners bool = true
	parsePairs  bool = false
	// readable bools 'shouldCreate'
	noFileNoService bool = false
	signupWelcome   bool = true
)

func grasp(ctx context.Context, cmd *cli.Command) error {
	kv, location := argsMonkeyBusiness(cmd, signupWelcome, parsePairs)
	access, err := bbolt.Open(location, 0o600, nil)
	errorutils.ExitOnFail(err, errorutils.WithMsg("monkey's hand wont open at "+location))
	for k, v := range kv {
		err = access.Batch(func(tx *bbolt.Tx) error {
			b, err := tx.CreateBucketIfNotExists([]byte{1})
			if err != nil {
				return err
			}
			return b.Put([]byte(k), []byte(v))
		})
		errorutils.WarnOnFail(err, errorutils.WithMsg("monkey won't grasp "+k+" at "+location))
	}

	return nil
}

func deliver(ctx context.Context, cmd *cli.Command) error {
	keys, location := argsMonkeyBusiness(cmd, noFileNoService, parseLoners)
	var values []string
	access, err := bbolt.Open(location, 0o600, nil)
	errorutils.ExitOnFail(err, errorutils.WithMsg("monkey's isn't playing games at "+location))
	err = access.View(func(tx *bbolt.Tx) error {
		for k := range keys {
			b := tx.Bucket([]byte{1})
			if b == nil {
				return errorutils.NewReport("monkey is malformed"+location, "b3VnxFdLPVk", errorutils.WithExitCode(1))
			}
			v := b.Get([]byte(k))
			if v == nil {
				return errorutils.NewReport("key not found: "+k, "AtAEgo5txim", errorutils.WithExitCode(1))
			}
			values = append(values, string(v))
		}

		return nil
	})
	errorutils.ExitOnFail(err)
	fmt.Print(strings.Join(values, "\n"))
	return nil
}

func hurl(ctx context.Context, cmd *cli.Command) error {
	keys, location := argsMonkeyBusiness(cmd, noFileNoService, parseLoners)
	access, err := bbolt.Open(location, 0o600, nil)
	errorutils.ExitOnFail(err, errorutils.WithMsg("monkey's keeping it for himself at "+location))

	return access.Batch(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte{1})
		if b == nil {
			return errorutils.NewReport("monkey is malformed"+location, "b3VnxFdLPVk", errorutils.WithExitCode(1))
		}
		for k := range keys {
			// TODO: should hurl print key one last time?
			b.Delete([]byte(k))
		}
		return nil
	})
}

// messy-argument parsing. to be able to handle spaces in the key@value pairs and have optional filename at the end
func argsMonkeyBusiness(cmd *cli.Command, shouldCreate bool, keysonly bool) (map[string]string, string) {
	location := `~/.monkey.bbolt`
	keyVals, args := parseArgs(cmd.Args().Slice())
	// check if last arg is a file
	if len(args) > 0 {
		isFilename, err := regexp.MatchString(`^[^\s]+\.bbolt$`, args[len(args)-1])
		errorutils.ExitOnFail(err, errorutils.WithMsg("Failed pattern matching optional filename"))
		if isFilename {
			location = args[len(args)-1]
			args = args[:len(args)-1]
		} else if !keysonly {
			logrus.Fatal("Monkey can't hold unpaired items right now: Offending argument: " + args[len(args)-1])
		}

	}
	{ // scoping
		usr, _ := user.Current()
		home := usr.HomeDir
		if location == "~" {
			// In case of "~", which won't be caught by the "else if"
			location = home
		} else if strings.HasPrefix(location, "~/") {
			// Use strings.HasPrefix so we don't match paths like
			// "/something/~/something/"
			location = filepath.Join(home, location[2:])
		}
	}
	// creating file
	if info, err := os.Stat(location); err == nil {
		if info.IsDir() {
			logrus.Fatal("Monkey can't play with directories")
		}
	} else if os.IsNotExist(err) {
		if shouldCreate {
			logrus.Warn("Creating " + location)
			err = nil
			_, err = os.Create(location)
			errorutils.ExitOnFail(err, errorutils.WithMsg("Could not create "+location))
		} else {
			logrus.Fatal("Monkey needs a file to play with")
		}
	} else {
		errorutils.ExitOnFail(err, errorutils.WithMsg("Could not stat "+location))
	}

	if !keysonly && len(keyVals) == 0 {
		logrus.Fatal("An empty request confuses monkey")
	} else if keysonly && len(keyVals) > 0 {
		logrus.Fatal("Providing values when you ask for keys is pointless")
	}
	if keysonly {
		for _, k := range args {
			keyVals[k] = ""
		}
	}
	return keyVals, location
}

// reassembleTokens joins parts split by spaces — ensuring balanced single quotes.
func reassembleTokens(args []string) []string {
	var tokens []string
	var current []string
	quoteCount := 0
	for _, token := range args {
		current = append(current, token)
		quoteCount += strings.Count(token, "'")
		if quoteCount%2 == 0 { // finished a quoted group
			tokens = append(tokens, strings.Join(current, " "))
			current = nil
			quoteCount = 0
		}
	}
	if len(current) > 0 {
		tokens = append(tokens, strings.Join(current, " "))
	}
	return tokens
}

// parseArgs splits each token on the first "@" — stripping any single quotes.
func parseArgs(args []string) (map[string]string, []string) {
	kvMap := make(map[string]string)
	var unpaired []string
	for _, token := range reassembleTokens(args) {
		if strings.Contains(token, "@") {
			parts := strings.SplitN(token, "@", 2)
			key := strings.Trim(parts[0], "'")
			value := strings.Trim(parts[1], "'")
			kvMap[key] = value
		} else {
			unpaired = append(unpaired, strings.Trim(token, "'"))
		}
	}
	var res []string
	if len(unpaired) > 0 {
		for _, a := range unpaired {
			if a != "" {
				res = append(res, a)
			}
		}
	}
	return kvMap, res
}
