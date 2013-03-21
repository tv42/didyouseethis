// TODO reuse these from twackup?

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

func IdFromTweet(tweet map[string]interface{}) (id uint64, err error) {
	id_raw := tweet["id_str"]
	id_str, ok := id_raw.(string)
	if !ok {
		msg := fmt.Sprintf("tweet id is not a string: %v", id_raw)
		err = errors.New(msg)
		return
	}

	id, err = strconv.ParseUint(id_str, 10, 64)
	if err != nil {
		return
	}
	return
}

func IsRetweet(tweet map[string]interface{}) bool {
	// we assume that if it's there, it's not empty, and vice versa
	_, ok := tweet["retweeted_status"]
	return ok
}

func SaveTweet(archive_dir string, retweet_dir string, tweet map[string]interface{}) (id uint64, err error) {
	// clean up Twitter's mistakes
	delete(tweet, "id")

	id, err = IdFromTweet(tweet)
	if err != nil {
		return
	}
	out, err := json.MarshalIndent(tweet, "", "  ")
	if err != nil {
		return
	}

	// roundtrip it from number back to string to canonicalize it
	id_str := strconv.FormatUint(id, 10)
	archive_path := filepath.Join(archive_dir, id_str+".json")
	retweet_path := filepath.Join(retweet_dir, id_str+".json")
	tmp := filepath.Join(archive_dir, id_str+"."+strconv.Itoa(os.Getpid())+".tmp")

	f, err := os.Create(tmp)
	if err != nil {
		return
	}
	_, err = f.Write(out)
	if err != nil {
		_ = os.Remove(tmp)
		return
	}
	err = f.Close()
	if err != nil {
		_ = os.Remove(tmp)
		return
	}

	if IsRetweet(tweet) {
		// don't even try retweeting retweets. twitter refuses
		// to do that, and while we could extract the original
		// and try to retweet that, doing that more than once
		// means we get hard-to-categorize errors and end up
		// leaving too many <id>.json.fail files around.
		// hopefully we'll see the original in the stream.
		log.Printf("not retweeting a retweet: %d", id)
	} else {
		// mark it to be retweeted first (at-least-once semantics)
		err = os.Link(tmp, retweet_path)
		// having an unprocessed to-retweet entry from an earlier run
		// (that failed to archive it) is ok; it should have the same
		// content
		if err != nil && !myIsExist(err) {
			_ = os.Remove(tmp)
			return
		}
	}

	err = os.Rename(tmp, archive_path)
	if err != nil {
		_ = os.Remove(tmp)
		return
	}
	return
}
