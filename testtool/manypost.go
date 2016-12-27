package main

import (
	"sync"

	"github.com/178inaba/tweeraser/config"
	"github.com/ChimeraCoder/anaconda"
	log "github.com/Sirupsen/logrus"
	uuid "github.com/satori/go.uuid"
)

const (
	configFilePath = "../etc/config.toml"
	oneLoopPostCnt = 10
)

func main() {
	api, err := newAPI(configFilePath)
	if err != nil {
		log.Fatal(err)
	}

	for {
		isErrCh := make(chan bool, oneLoopPostCnt)
		wg := new(sync.WaitGroup)
		for i := 0; i < oneLoopPostCnt; i++ {
			wg.Add(1)
			go postTweet(api, wg, isErrCh)
		}

		wg.Wait()
		close(isErrCh)

		for isErr := range isErrCh {
			if isErr {
				log.Fatal("An error occurred.")
			}
		}
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

func postTweet(api *anaconda.TwitterApi, wg *sync.WaitGroup, isErrCh chan<- bool) {
	defer wg.Done()

	tweet := uuid.NewV4().String()
	t, err := api.PostTweet(tweet, nil)
	if err != nil {
		log.WithField("tweet", tweet).Error(err)
		isErrCh <- true
		return
	}

	log.WithFields(log.Fields{"id": t.Id, "tweet": t.Text}).Infof("Success!")
	isErrCh <- false
}
