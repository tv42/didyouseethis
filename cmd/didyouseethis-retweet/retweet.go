package main

import (
	"encoding/json"
	"fmt"
	"github.com/alloy-d/goauth"
	"mime"
)

type twitterError struct {
	// https://code.google.com/p/twitter-api/issues/detail?id=1333
	Errors string
}

type TwitterError struct {
	StatusCode int
	Status     string
	Message    string
}

func (e TwitterError) Error() string {
	s := fmt.Sprintf("twitter error: %s", e.Status)
	if e.Message != "" {
		s = fmt.Sprintf("%s: %s", s, e.Message)
	}
	return s
}

func ErrIsPermanent(err error) bool {
	err2, ok := err.(TwitterError)
	if !ok {
		return false
	}
	return err2.StatusCode == 403 &&
		err2.Message == "sharing is not permissible for this status (Share validations failed)"
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

		mediatype, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))

		if err == nil && mediatype == "application/json" {
			dec := json.NewDecoder(response.Body)
			_ = dec.Decode(&extra)
		}
		return TwitterError{
			StatusCode: response.StatusCode,
			Status:     response.Status,
			Message:    extra.Errors,
		}
	}
	return nil
}
