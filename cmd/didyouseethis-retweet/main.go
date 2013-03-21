package main

import (
	"flag"
	"fmt"
	"github.com/cznic/sortutil"
	"github.com/tv42/didyouseethis"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s CONFIG.YAML DIR\n", os.Args[0])
	flag.PrintDefaults()
}

func maybeMkdir(path string) error {
	err := os.Mkdir(path, 0755)
	if os.IsExist(err) {
		err = nil
	}
	return nil
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if len(flag.Args()) != 2 {
		flag.Usage()
		os.Exit(1)
	}

	config_path := flag.Args()[0]
	state_dir := flag.Args()[1]

	config, err := didyouseethis.ReadConfig(config_path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot read config: %s\n", os.Args[0], err)
		os.Exit(1)
	}

	retweet_dir := filepath.Join(state_dir, "retweet")
	err = maybeMkdir(retweet_dir)
	if err != nil {
		log.Fatalf("cannot mkdir: %s", err)
	}

	o, err := didyouseethis.NewAuth(config, state_dir)
	if err != nil {
		log.Fatalf("cannot prepare OAuth: %v", err)
	}

	dir, err := os.Open(retweet_dir)
	if err != nil {
		log.Fatalf("cannot open state dir: %v", err)
	}
	defer dir.Close()

	fmt.Printf("Starting to retweet...\n")
	for {
		var ids sortutil.Uint64Slice

		for {
			names, err := dir.Readdirnames(1000)
			if err != nil && err != io.EOF {
				log.Fatalf("cannot read state dir: %v", err)
			}

			if len(names) == 0 {
				break
			}

			for _, name := range names {
				// all we need for retweeting is the id, so don't bother decoding the json
				ext := filepath.Ext(name)
				if ext != ".json" {
					continue
				}
				id_str := name[:len(name)-len(ext)]

				// decode the number to be able to sort
				id, err := strconv.ParseUint(id_str, 10, 64)
				if err != nil {
					continue
				}

				ids = append(ids, id)
			}
		}

		if len(ids) == 0 {
			// nothing to do
			// TODO use inotify
			fmt.Printf("Sleeping...\n")
			time.Sleep(1 * time.Second)
			continue
		}

		sort.Sort(ids)

		for _, id := range ids {
			err := Retweet(id, o)
			if err != nil {
				log.Fatalf("can't retweet: %s", err)
			}
			id_str := strconv.FormatUint(id, 10)
			fmt.Printf("Retweeted: %s\n", id_str)
			err = os.Remove(filepath.Join(retweet_dir, id_str+".json"))
			if err != nil {
				log.Fatalf("can't remove retweet file: %s", err)
			}
		}
	}
}
