package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/alloy-d/goauth"
	"github.com/tv42/didyouseethis"
	"io"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s CONFIG.YAML DIR\n", os.Args[0])
	flag.PrintDefaults()
}

// Merge multiple simple search strings into a comma-separated one.
func merge_keywords(keywords []string) string {
	// TODO is there a max len?
	return strings.Join(keywords, ",")
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

	oauth_path := filepath.Join(state_dir, ".oauth")

	archive_dir := filepath.Join(state_dir, "archive")
	err = maybeMkdir(archive_dir)
	if err != nil {
		log.Fatalf("cannot mkdir: %s", err)
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

	track := merge_keywords(config.Keywords)
	url := "https://stream.twitter.com/1.1/statuses/filter.json"
	response, err := o.Post(
		url,
		map[string]string{
			"track": track,
		})
	if err != nil {
		log.Fatalf("can't stream: %s", err)
	}
	if response.StatusCode != 200 {
		log.Fatalf("can't stream: %s", response.Status)
	}

	ctype := response.Header.Get("content-type")
	mediatype, _, err := mime.ParseMediaType(ctype)
	if err != nil {
		log.Fatalf("stream content-type is broken: %q", ctype)
	}
	if mediatype != "application/json" {
		log.Fatalf("stream is not json: %q", ctype)
	}

	fmt.Printf("Starting to stream...\n")
	dec := json.NewDecoder(response.Body)
	for {
		var msg map[string]interface{}
		err := dec.Decode(&msg)
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatalf("error decoding json: %s\n", err)
			break
		}

		switch {
		//TODO handle limit
		//TODO handle status_withheld
		//TODO handle user_withheld
		//TODO handle disconnect

		case msg["text"] != nil:
			fmt.Printf("got tweet: %q\n", msg["text"])

			_, ok := msg["id_str"]
			if !ok {
				panic(fmt.Sprintf("TODO no id: %v", msg))
			}

			_, err := SaveTweet(archive_dir, retweet_dir, msg)
			if err != nil {
				log.Fatalf("can't save tweet: %s", err)
			}

		default:
			fmt.Printf("Unhandled message type: %q\n", msg)
		}
	}
}
