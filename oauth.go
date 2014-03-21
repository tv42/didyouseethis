package didyouseethis

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alloy-d/goauth"
)

// Note: this function interacts with the user.
func NewAuth(config *Config, state_dir string) (*oauth.OAuth, error) {
	o := new(oauth.OAuth)
	o.ConsumerKey = config.OAuth.Key
	o.ConsumerSecret = config.OAuth.Secret

	o.RequestTokenURL = "https://api.twitter.com/oauth/request_token"
	o.OwnerAuthURL = "https://api.twitter.com/oauth/authorize"
	o.AccessTokenURL = "https://api.twitter.com/oauth/access_token"

	o.SignatureMethod = oauth.HMAC_SHA1

	oauth_path := filepath.Join(state_dir, ".oauth")

	err := o.Load(oauth_path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	if o.AccessToken == "" {
		err := o.GetRequestToken()
		if err != nil {
			return nil, err
		}

		url, err := o.AuthorizationURL()
		if err != nil {
			return nil, err
		}

		fmt.Printf("Please authorize this app at:\n\n  %s\n"+
			"\nand enter the PIN here: ", url)
		var verifier string
		fmt.Scanln(&verifier)

		err = o.GetAccessToken(verifier)
		if err != nil {
			return nil, err
		}

		err = o.Save(oauth_path)
		if err != nil {
			return nil, err
		}
	}

	return o, nil
}
