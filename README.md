didyouseethis -- Retweet anything with certain keywords
=======================================================

This service will allow you to run a Twitter bot that retweets
anything matching its criteria. One way of using it would be to have a
protected account retweet things you might be interested in, and then
letting select people follow that protected account.

To install, you'll need a working Go setup. Then run:

    go get github.com/tv42/didyouseethis/cmd/didyouseethis-save
    go get github.com/tv42/didyouseethis/cmd/didyouseethis-retweet

And now your GOPATH's bin directory should have the commands.

To use, you'll need to register a Twitter app, and put its key in a
YAML config file, along with the keywords you want to track:

    oauth:
      key: KEY_GOES_HERE
      secret: SECRET_GOES_HERE
    keywords:
    - orange juice
    - milk

For keywords, each word in a single line has to match; any item
matching means tweet is included.

Test them out by running:

    mkdir state
    didyouseethis-save MY_CONF_FILE.yaml state &
    didyouseethis-retweet MY_CONF_FILE.yaml state

On the first run, you will be asked to authorize the app; once the
authorization is ok, the commands will run in batch mode.

The above commands will exit on errors, and expect to be run under a
daemon supervisor that restarts them.

You are responsible for following Twitter's policies; there are plenty
of details where didyouseethis does not currently enforce correct
behavior.

The code is still uglier than it should be, but hey, it works!
