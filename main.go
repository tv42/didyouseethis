package main

import (
	"os"
	"fmt"
	"github.com/hoisie/twitterstream"
	"log"
	"github.com/alloy-d/goauth"
	"launchpad.net/goyaml"
	"flag"
	"io/ioutil"
	"errors"
	"strings"
)

type Config struct {
	User string
	Password string
	OAuth struct {
		Key string
		Secret string
	}
	Keywords []string
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s CONFIG.YAML OAUTH_STATE_FILE\n", os.Args[0])
	flag.PrintDefaults()
}

func readConfig(path string) (*Config, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config Config
	err = goyaml.Unmarshal(buf, &config)
	if err != nil {
		return nil, err
	}

	// validate that required fields were set
	if config.User == "" {
		return nil, errors.New("missing field: user")
	}
	if config.Password == "" {
		return nil, errors.New("missing field: password")
	}
	if config.OAuth.Key == "" {
		return nil, errors.New("missing field: oauth key")
	}
	if config.OAuth.Secret == "" {
		return nil, errors.New("missing field: oauth secret")
	}
	if len(config.Keywords) == 0 {
		return nil, errors.New("missing field: keywords")
	}

	return &config, nil
}

// Merge multiple simple search strings into a comma-separated one.
func merge_keywords(keywords []string) []string {
	// TODO is there a max len?
	return []string{
		strings.Join(keywords, ","),
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
	oauth_path := flag.Args()[1]

	config, err := readConfig(config_path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot read config: %s\n", os.Args[0], err)
		os.Exit(1)
	}

	stream := make(chan *twitterstream.Tweet)
	// TODO use oauth for streaming too, only if twitter supported it
	client := twitterstream.NewClient(config.User, config.Password)

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
	err = client.Track(track, stream)
	if err != nil {
		log.Fatalf("twitter stream track failed: %s", err)
	}
	for {
		tw := <-stream
		fmt.Printf("%s: %s\n", tw.User.Screen_name, tw.Text)

		url := fmt.Sprintf(
			"https://api.twitter.com/1/statuses/retweet/%d.json",
			tw.Id,
		)
		response, err := o.Post(
			url,
			map[string]string{
				"trim_user": "true",
			})
		if err != nil {
			log.Fatalf("can't retweet: %s", err)
		}
		if response.StatusCode != 200 {
			log.Fatalf("can't retweet: %s", response.Status)
		}
		fmt.Printf("Retweeted! %+v", response)
	}
}
