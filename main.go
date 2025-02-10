package main

import (
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/pydpll/errorutils"
	"github.com/sirupsen/logrus"
	"go.etcd.io/bbolt"
)

//TODO: with flag change presntation to colon separated path
//TODO: check why terinator is not bolding

var (
	shouldCount *bool
	CommitID    string
)

func main() {
	shouldCount = flag.Bool("count", false, "Enable count mode: total number of leaves is printed instead of listed.")
	help := flag.Bool("help", false, "Show this help. Halts execution.")
	flag.BoolVar(shouldCount, "c", false, "Shorthand for --count")
	flag.Parse()
	if *help {
		//multiline string
		x := "version " + CommitID + "\n" + `Usage: monkey [OPTIONS] file1 file2 ...

Options:
`
		fmt.Println(x)
		flag.Usage()
		os.Exit(0)
	}
	if len(flag.Args()) == 0 {
		fmt.Println("Please enter a file name")
		fmt.Println("Usage: monkey [OPTIONS] file1 file2 ...")
		os.Exit(1)
	}
	for _, filex := range flag.Args() {
		exploreFile(filex)
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
		go treeWorker(db, TOPKEY_ch, RETURN_ch, &wg)
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

func colorize(text, color string) string {
	return "\033[" + color + "m" + text + "\033[0m"
}
