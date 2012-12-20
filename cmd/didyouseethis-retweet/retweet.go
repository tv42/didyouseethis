package main

import (
	"fmt"
	"github.com/alloy-d/goauth"
)

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
		return fmt.Errorf("twitter error: %s", response.Status)
	}
	return nil
}
