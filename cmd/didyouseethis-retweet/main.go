package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/alloy-d/goauth"
	"github.com/tv42/didyouseethis"
	"golang.org/x/exp/inotify"
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

func isWork(name string) (order uint64, ok bool) {
	ext := filepath.Ext(name)
	if ext != ".json" {
		return 0, false
	}
	order_str := name[:len(name)-len(ext)]

	// decode the number to be able to sort
	order, err := strconv.ParseUint(order_str, 10, 64)
	if err != nil {
		return 0, false
	}
	return order, true
}

func retweet_file(id uint64, name string, o *oauth.OAuth, retweet_dir string) {
	err := Retweet(id, o)
	if err != nil {
		if ErrIsPermanent(err) {
			// this is not a temporary error, so stop trying
			log.Printf("twitter refused retweet: %d: %v", id, err)
			err = os.Rename(filepath.Join(retweet_dir, name), filepath.Join(retweet_dir, name+".fail"))
			if err != nil {
				log.Fatalf("can't move failing tweet aside: %s", err)
			}
			return
		}
		log.Fatalf("can't retweet: %s", err)
	}
	fmt.Printf("Retweeted: %d\n", id)
	err = os.Remove(filepath.Join(retweet_dir, name))
	if err != nil {
		log.Fatalf("can't remove retweet file: %s", err)
	}
}

type workItem struct {
	id   uint64
	name string
}

type work []workItem

func (s work) Len() int           { return len(s) }
func (s work) Less(i, j int) bool { return s[i].id < s[j].id }
func (s work) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func retweet_old(dir *os.File, o *oauth.OAuth, retweet_dir string) {
	// consume what's there
	for {
		var pending work

		for {
			names, err := dir.Readdirnames(1000)
			if err != nil && err != io.EOF {
				log.Fatalf("cannot read work dir: %v", err)
			}

			if len(names) == 0 {
				break
			}

			for _, name := range names {
				// all we need for retweeting is the id, so don't bother decoding the json
				id, ok := isWork(name)
				if !ok {
					continue
				}
				pending = append(pending, workItem{id, name})
			}
		}

		if len(pending) == 0 {
			// we've emptied our work queue
			break
		}

		sort.Sort(pending)

		for _, item := range pending {
			retweet_file(item.id, item.name, o, retweet_dir)
		}
	}
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

	watch, err := inotify.NewWatcher()
	if err != nil {
		log.Fatalf("cannot use inotify: %v", err)
	}
	defer watch.Close()

	// start watching for file additions
	err = watch.AddWatch(retweet_dir, inotify.IN_CREATE)
	if err != nil {
		log.Fatalf("cannot watch state dir: %v", err)
	}

	fmt.Printf("Starting to retweet...\n")

	for {
		retweet_old(dir, o, retweet_dir)
		select {
		case err = <-watch.Error:
			log.Fatalf("error in watching: %v", err)
		case ev := <-watch.Event:
			switch {
			case ev.Mask&inotify.IN_Q_OVERFLOW != 0:
				// queue overflow, go back to readdir until empty
				continue
			case ev.Mask&inotify.IN_CREATE != 0:
				// inotify gives us full paths
				name := filepath.Base(ev.Name)
				id, ok := isWork(name)
				if !ok {
					continue
				}
				retweet_file(id, name, o, retweet_dir)
			}
		}
	}
}
