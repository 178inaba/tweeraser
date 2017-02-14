package main

import (
	"fmt"
	"net/url"
	"os"
	"sync"

	"github.com/178inaba/tweeraser/config"
	"github.com/178inaba/tweeraser/model"
	"github.com/178inaba/tweeraser/model/mysql"
	"github.com/ChimeraCoder/anaconda"
	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
)

const configFilePath = "etc/config.toml"

type client struct {
	api *anaconda.TwitterApi
	ets model.EraseTweetService
}

func main() {
	api, err := newAPI(configFilePath)
	if err != nil {
		log.Fatal(err)
	}

	ets, err := newEraseTweetService()
	if err != nil {
		log.Warn(err)
	}

	c := client{api: api, ets: ets}

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
			go c.deleteTweet(tweet.Id, wg, isErrCh)
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

func (c client) deleteTweet(id int64, wg *sync.WaitGroup, isErrCh chan<- bool) {
	defer wg.Done()

	l := log.WithField("id", id)

	t, err := c.api.DeleteTweet(id, true)
	if err != nil {
		l.Errorf("Fail delete: %v", err)
		isErrCh <- true
		return
	}

	insertID, err := c.insert(t)
	if err != nil {
		l.Errorf("Fail insert: %v", err)
		isErrCh <- true
		return
	} else if insertID != 0 {
		l = l.WithField("insert_id", insertID)
	}

	l.Info("Delete success!")
	isErrCh <- false
}

func (c client) insert(t anaconda.Tweet) (uint64, error) {
	var insertID uint64
	if c.ets != nil {
		postedAt, err := t.CreatedAtTime()
		et := &model.EraseTweet{
			TwitterTweetID: uint64(t.Id), Tweet: t.Text, PostedAt: postedAt}
		insertID, err = c.ets.Insert(et)
		if err != nil {
			return 0, err
		}
	}

	return insertID, nil
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

func newEraseTweetService() (model.EraseTweetService, error) {
	db, err := mysql.Open("root", "", "tweeraser")
	if err == nil {
		if db.Ping() == nil {
			return mysql.NewEraseTweetService(db), nil
		}

		return nil, errors.Errorf("Fail db ping: %s.", err)
	}

	return nil, errors.Errorf("Fail db open: %s.", err)
}
