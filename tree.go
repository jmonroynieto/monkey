package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/pydpll/errorutils"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
	"go.etcd.io/bbolt"
)

func readFiles(ctx context.Context, cmd *cli.Command) error {
	if len(cmd.Args().Slice()) == 0 {
		redMSG := colorize("Please enter a file name", "31")
		logrus.Error(redMSG)
	}
	worker = treeWorker
	if altPrint {
		worker = trailWorker
	}

	for _, file := range cmd.Args().Slice() {
		exploreFile(file)
	}
	return nil
}

func exploreFile(name string) {
	if _, err := os.Stat(name); os.IsNotExist(err) {
		redMSG := colorize("Please enter a file name", "31")
		logrus.Error(redMSG + ":" + name)
	}
	db, err := bbolt.Open(name, 0o600, nil)
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
			if !shouldCount {
				out += fmt.Sprintf("%s└── %s\n", prefix, colorize(string(k), "1"))
			}
		} else {
			subBucket := b.Bucket(k)
			subText, subCount := tree(subBucket, prefix+"    ")
			if shouldCount {
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
			if shouldCount && c > 0 {
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
			out += fmt.Sprintf("\n%s:%s", prefix, colorize(string(k), "1"))
		} else {
			subBucket := b.Bucket(k)
			out += trail(subBucket, prefix+":"+string(k))
		}
		return nil
	})
	return out[1:]
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
