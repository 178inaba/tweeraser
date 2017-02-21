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

type tweetEraseClient struct {
	api *anaconda.TwitterApi
	ets model.EraseTweetService
}

func main() {
	os.Exit(run())
}

func run() int {
	api, err := newAPI(configFilePath)
	if err != nil {
		log.Error(err)
		return 1
	}

	ets, err := newEraseTweetService()
	if err != nil {
		log.Warn(err)
	}

	c := tweetEraseClient{api: api, ets: ets}

	err = c.eraseTimeline()
	if err != nil {
		log.Error(err)
		return 1
	}

	return 0
}

func (c tweetEraseClient) eraseTimeline() error {
	id, err := c.getMyUserID()
	if err != nil {
		return err
	}

	v := url.Values{}
	v.Set("user_id", fmt.Sprint(id))
	v.Set("count", fmt.Sprint(200))
	v.Set("trim_user", "true")
	v.Set("contributor_details", "false")
	v.Set("include_rts", "false")

	for {
		tweets, err := c.api.GetUserTimeline(v)
		if err != nil {
			return err
		} else if len(tweets) == 0 {
			return nil
		}

		ids := make([]int64, len(tweets))
		for i, t := range tweets {
			ids[i] = t.Id
		}

		c.eraseIDs(ids)

		v.Set("max_id", fmt.Sprint(tweets[len(tweets)-1].Id-1))
	}
}

func (c tweetEraseClient) eraseIDs(ids []int64) error {
	isErrCh := make(chan bool, len(ids))
	wg := new(sync.WaitGroup)
	for _, id := range ids {
		wg.Add(1)
		go c.eraseTweet(id, wg, isErrCh)
	}

	wg.Wait()
	close(isErrCh)

	for isErr := range isErrCh {
		if isErr {
			return errors.New("an error occurred")
		}
	}

	return nil
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

func (c tweetEraseClient) eraseTweet(id int64, wg *sync.WaitGroup, isErrCh chan<- bool) {
	defer wg.Done()

	l := log.WithField("id", id)

	t, err := c.api.DeleteTweet(id, true)
	if err != nil {
		l.Errorf("Fail erase: %v", err)
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

	l.Info("Successfully erased!")
	isErrCh <- false
}

func (c tweetEraseClient) insert(t anaconda.Tweet) (uint64, error) {
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

func (c tweetEraseClient) getMyUserID() (int64, error) {
	v := url.Values{}
	v.Set("include_entities", "false")
	v.Set("skip_status", "true")
	v.Set("include_email", "false")

	u, err := c.api.GetSelf(v)
	if err != nil {
		return 0, err
	}

	return u.Id, nil
}

func newEraseTweetService() (model.EraseTweetService, error) {
	db, err := mysql.Open("root", "", "tweeraser")
	if err != nil {
		return nil, errors.Errorf("Fail db open: %s.", err)
	}

	if err := db.Ping(); err != nil {
		return nil, errors.Errorf("Fail db ping: %s.", err)
	}

	return mysql.NewEraseTweetService(db), nil
}
