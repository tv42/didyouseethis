package main

import (
	"flag"
	"fmt"
	"github.com/alloy-d/goauth"
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
	fmt.Fprintf(os.Stderr, "Usage: %s CONFIG.YAML OAUTH_STATE_FILE DIR\n", os.Args[0])
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

	if len(flag.Args()) != 3 {
		flag.Usage()
		os.Exit(1)
	}

	config_path := flag.Args()[0]
	oauth_path := flag.Args()[1]
	state_dir := flag.Args()[2]

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

	o := new(oauth.OAuth)
	o.ConsumerKey = config.OAuth.Key
	o.ConsumerSecret = config.OAuth.Secret

	o.RequestTokenURL = "https://api.twitter.com/oauth/request_token"
	o.OwnerAuthURL = "https://api.twitter.com/oauth/authorize"
	o.AccessTokenURL = "https://api.twitter.com/oauth/access_token"

	o.SignatureMethod = oauth.HMAC_SHA1

	err = o.Load(oauth_path)
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("Error loading OAuth information: %s", err)
	}

	if o.AccessToken == "" {
		log.Printf("gonna ask for token\n")
		err := o.GetRequestToken()
		if err != nil {
			log.Fatalf("get request token: %s", err)
		}

		url, err := o.AuthorizationURL()
		if err != nil {
			log.Fatalf("authorization url: %s", err)
		}

		fmt.Printf("Please authorize this app at:\n\n  %s\n"+
			"\nand enter the PIN here: ", url)
		var verifier string
		fmt.Scanln(&verifier)

		log.Printf("got it! %q\n", verifier)

		err = o.GetAccessToken(verifier)
		if err != nil {
			log.Fatalf("get access token: %s", err)
		}

		err = o.Save(oauth_path)
		if err != nil {
			log.Fatalf("Error saving OAuth information: %s", err)
		}
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
