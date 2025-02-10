package main

import (
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/jxskiss/mcli"
	"github.com/pydpll/errorutils"
	"github.com/sirupsen/logrus"
	"go.etcd.io/bbolt"
)

var (
	CommitID    string
	worker      workers
	shouldCount *bool
	files       []string
)

type workers func(db *bbolt.DB, KEYS_ch <-chan string, RETURN_ch chan<- string, wg *sync.WaitGroup)

func main() {
	var Args struct {
		Count    bool `cli:"-c, --count, change tree view to not show leaves"`
		AltPrint bool `cli:"-p, --altPrint, change tree view to colon separated path"`
	}
	// those that start with flags
	flags := []string{}
	// those that start with files
	for _, arg := range os.Args[1:] {
		if arg[0] == '-' {
			flags = append(flags, arg)
		} else {
			files = append(files, arg)
		}
	}
	mcli.AddHelp()
	mcli.AddRoot(readFiles)
	shouldCount = &Args.Count
	_, err := mcli.Parse(&Args, mcli.WithErrorHandling(flag.ContinueOnError), mcli.WithArgs(flags))
	errorutils.ExitOnFail(err)
	worker = treeWorker
	if Args.AltPrint {
		worker = trailWorker
	}
	mcli.Run()
}

func readFiles() {
	if len(files) == 0 {
		fmt.Println("Please enter a file name")
		mcli.PrintHelp()
		os.Exit(1)
	}
	for _, file := range files {
		exploreFile(file)
	}
}

func exploreFile(name string) {

	if _, err := os.Stat(name); os.IsNotExist(err) {
		redMSG := colorize("Please enter a file name", "31")
		logrus.Error(redMSG + ":" + name)
	}
	db, err := bbolt.Open(name, 0600, nil)
	errorutils.ExitOnFail(err)
	defer db.Close()

	TOPKEY_ch := make(chan string)
	RETURN_ch := make(chan string)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go worker(db, TOPKEY_ch, RETURN_ch, &wg)
	}

	go func() {
		_ = db.View(func(tx *bbolt.Tx) error {
			return tx.ForEach(func(name []byte, b *bbolt.Bucket) error {
				TOPKEY_ch <- string(name)
				return nil
			})
		})
		close(TOPKEY_ch)
	}()

	go func() {
		wg.Wait()
		close(RETURN_ch)
	}()

	for res := range RETURN_ch {
		fmt.Println(res)
	}
}

func tree(b *bbolt.Bucket, prefix string) (string, int) {
	var out string
	totalCount := 0
	b.ForEach(func(k, v []byte) error {
		if v != nil {
			totalCount++
			if !*shouldCount {
				out += fmt.Sprintf("%s└── %s\n", prefix, colorize(string(k), "1"))
			}
		} else {
			subBucket := b.Bucket(k)
			subText, subCount := tree(subBucket, prefix+"    ")
			if *shouldCount {
				out += fmt.Sprintf("%s└── %s (%d leaves)\n", prefix, string(k), subCount)
			} else {
				out += fmt.Sprintf("%s└── %s\n%s", prefix, string(k), subText)
			}
			totalCount += subCount
		}
		return nil
	})
	return out, totalCount
}

func treeWorker(db *bbolt.DB, KEYS_ch <-chan string, RETURN_ch chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	for key := range KEYS_ch {
		_ = db.View(func(tx *bbolt.Tx) error {
			b := tx.Bucket([]byte(key))
			if b == nil {
				RETURN_ch <- fmt.Sprintf("Bucket %s not found", key)
				return nil
			}
			o, c := tree(b, "")
			if *shouldCount && c > 0 {
				key += fmt.Sprintf(" (%d total leaves)", c)
			}
			RETURN_ch <- fmt.Sprintf("%s:\n%s", key, o)
			return nil
		})
	}
}

func trail(b *bbolt.Bucket, prefix string) string {
	var out string
	b.ForEach(func(k, v []byte) error {
		if v != nil {
			out += fmt.Sprintf("%s:%s\n", prefix, colorize(string(k), "1"))
		} else {
			subBucket := b.Bucket(k)
			out = trail(subBucket, prefix+":"+string(k))
		}
		return nil
	})
	return out
}

func trailWorker(db *bbolt.DB, KEYS_ch <-chan string, RETURN_ch chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	for key := range KEYS_ch {
		_ = db.View(func(tx *bbolt.Tx) error {
			b := tx.Bucket([]byte(key))
			if b == nil {
				RETURN_ch <- fmt.Sprintf("Bucket %s not found", key)
				return nil
			}

			RETURN_ch <- trail(b, key)
			return nil
		})
	}
}

func colorize(text, color string) string {
	return "\033[" + color + "m" + text + "\033[0m"
}
