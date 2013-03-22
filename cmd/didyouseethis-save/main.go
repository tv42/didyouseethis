package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/tv42/didyouseethis"
	"github.com/tv42/didyouseethis/watchdog"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"
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

	o, err := didyouseethis.NewAuth(config, state_dir)
	if err != nil {
		log.Fatalf("cannot prepare OAuth: %v", err)
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
	body := bufio.NewReader(response.Body)

	dog := watchdog.New(90 * time.Second)
	lines := make(chan []byte)
	readError := make(chan error, 1)
	go func() {
		defer close(lines)
		defer close(readError)
		for {
			line, err := body.ReadBytes('\n')
			if err != nil {
				readError <- err
				break
			}
			lines <- line
		}
	}()

	for {
		select {
		case line, ok := <-lines:
			if !ok {
				err = <-readError
				log.Fatalf("error reading stream: %s\n", err)
			}

			dog.Pet()

			if bytes.Equal(line, []byte{'\r', '\n'}) {
				continue
			}

			var msg map[string]interface{}
			err = json.Unmarshal(line, &msg)
			if err != nil {
				log.Fatalf("bad json from stream: %v", err)
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

		case <-dog.Bark:
			log.Fatalf("stream timeout")
		}
	}
}
