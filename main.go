package main

import (
	"fmt"
	"net/url"
	"os"
	"sync"

	"github.com/178inaba/tweeraser/config"
	"github.com/ChimeraCoder/anaconda"
	log "github.com/Sirupsen/logrus"
)

const configFilePath = "etc/config.toml"

func main() {
	api, err := newAPI(configFilePath)
	if err != nil {
		log.Fatal(err)
	}

	id, err := getMyUserID(api)
	if err != nil {
		log.Fatal(err)
	}

	v := url.Values{}
	v.Set("user_id", fmt.Sprint(id))
	v.Set("count", fmt.Sprint(200))
	v.Set("trim_user", "true")
	v.Set("contributor_details", "false")
	v.Set("include_rts", "false")

	for {
		tweets, err := api.GetUserTimeline(v)
		if err != nil {
			log.Fatal(err)
		} else if len(tweets) == 0 {
			os.Exit(0)
			return
		}

		isErrCh := make(chan bool, len(tweets))
		wg := new(sync.WaitGroup)
		for _, tweet := range tweets {
			wg.Add(1)
			go deleteTweet(api, tweet.Id, wg, isErrCh)
		}

		wg.Wait()
		close(isErrCh)

		for isErr := range isErrCh {
			if isErr {
				log.Fatal("An error occurred.")
			}
		}

		v.Set("max_id", fmt.Sprint(tweets[len(tweets)-1].Id-1))
	}
}

func newAPI(path string) (*anaconda.TwitterApi, error) {
	conf, err := config.LoadConfig(path)
	if err != nil {
		return nil, err
	}

	anaconda.SetConsumerKey(conf.ConsumerKey)
	anaconda.SetConsumerSecret(conf.ConsumerSecret)
	return anaconda.NewTwitterApi(conf.AccessToken, conf.AccessTokenSecret), nil
}

func deleteTweet(api *anaconda.TwitterApi, id int64, wg *sync.WaitGroup, isErrCh chan<- bool) {
	defer wg.Done()

	l := log.WithField("id", id)

	_, err := api.DeleteTweet(id, true)
	if err != nil {
		l.Errorf("Delete fail: %v", err)
		isErrCh <- true
		return
	}

	l.Info("Delete success!")
	isErrCh <- false
}

func getMyUserID(api *anaconda.TwitterApi) (int64, error) {
	v := url.Values{}
	v.Set("include_entities", "false")
	v.Set("skip_status", "true")
	v.Set("include_email", "false")

	u, err := api.GetSelf(v)
	if err != nil {
		return 0, err
	}

	return u.Id, nil
}
