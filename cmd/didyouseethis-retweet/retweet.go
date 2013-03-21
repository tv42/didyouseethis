package main

import (
	"encoding/json"
	"fmt"
	"github.com/alloy-d/goauth"
	"log"
)

type twitterError struct {
	// https://code.google.com/p/twitter-api/issues/detail?id=1333
	Errors string
}

func Retweet(id uint64, o *oauth.OAuth) error {
	url := fmt.Sprintf(
		"https://api.twitter.com/1.1/statuses/retweet/%d.json",
		id,
	)
	response, err := o.Post(
		url,
		map[string]string{
			"trim_user": "true",
		})
	if err != nil {
		return err
	}
	if response.StatusCode != 200 {
		var extra twitterError
		if response.Header.Get("Content-Type") == "application/json; charset=utf-8" {
			dec := json.NewDecoder(response.Body)
			_ = dec.Decode(&extra)
		}
		if response.StatusCode == 403 && extra.Errors == "sharing is not permissible for this status (Share validations failed)" {
			// this is not a temporary error, so stop trying
			log.Printf("twitter refused retweet: %d", id)
			return nil
		}
		return fmt.Errorf("twitter error: %s: %q", response.Status, extra.Errors)
	}
	return nil
}
