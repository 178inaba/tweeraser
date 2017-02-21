package main

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"sync"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/178inaba/tweeraser/config"
	"github.com/178inaba/tweeraser/model"
	"github.com/178inaba/tweeraser/model/mysql"
	"github.com/ChimeraCoder/anaconda"
	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
)

const configFilePath = "etc/config.toml"

var (
	csvFilePath = kingpin.Flag("csv-file", "all tweets csv file (tweets.csv) path.").String()
	zipFilePath = kingpin.Flag("zip-file", "all tweets zip file path.").String()
	isNoCheck   = kingpin.Flag("no-check", "no check csv or zip.").Bool()
)

func main() {
	kingpin.Parse()
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

	if *csvFilePath != "" {
		err = c.eraseCsv()
	} else if *zipFilePath != "" {
		err = c.eraseZip()
	} else {
		err = c.eraseTimeline()
	}
	if err != nil {
		log.Error(err)
		return 1
	}

	return 0
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

type tweetEraseClient struct {
	api *anaconda.TwitterApi
	ets model.EraseTweetService
}

func (c tweetEraseClient) eraseCsv() error {
	f, err := os.Open(*csvFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	return c.eraseCsvReader(f)
}

func (c tweetEraseClient) eraseZip() error {
	r, err := zip.OpenReader(*zipFilePath)
	if err != nil {
		return err
	}
	defer r.Close()

	var zf *zip.File
	for _, f := range r.File {
		if f.Name == "tweets.csv" {
			zf = f
			break
		}
	}

	rc, err := zf.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	return c.eraseCsvReader(rc)
}

func (c tweetEraseClient) eraseCsvReader(r io.Reader) error {
	cr := csv.NewReader(r)
	header, err := cr.Read()
	if err != nil {
		return err
	}

	var tweetIDIndex int
	for i, name := range header {
		if name == "tweet_id" {
			tweetIDIndex = i
			break
		}
	}

	var ids []int64
	for {
		record, err := cr.Read()
		if err == io.EOF {
			if *isNoCheck {
				return c.eraseIDs(ids)
			}

			return c.checkBeforeEraseIDs(ids)
		} else if err != nil {
			return err
		}

		id, err := strconv.ParseInt(record[tweetIDIndex], 10, 64)
		if err != nil {
			return err
		}

		ids = append(ids, id)
	}
}

func (c tweetEraseClient) checkBeforeEraseIDs(ids []int64) error {
	var idsPackets [][]int64
	packetSize := 100
	for packetSize > 0 {
		idsPackets = append(idsPackets, ids[:packetSize])
		ids = append(ids[:0], ids[packetSize:]...)
		checkIDsLen := len(ids)
		if checkIDsLen < 100 {
			packetSize = checkIDsLen
		}
	}

	log.Info("Checking...")
	receiveIDsCh := make(chan []int64, len(idsPackets))
	wg := new(sync.WaitGroup)
	for i, ip := range idsPackets {
		wg.Add(1)
		go c.checkIDs(ip, receiveIDsCh, wg, i)
	}

	wg.Wait()
	log.Info("Done!")
	close(receiveIDsCh)

	var validIDs []int64
	for receiveIDs := range receiveIDsCh {
		validIDs = append(validIDs, receiveIDs...)
	}

	return c.eraseIDs(validIDs)
}

func (c tweetEraseClient) checkIDs(ids []int64, validIDsCh chan<- []int64, wg *sync.WaitGroup, goroutineIndex int) {
	defer wg.Done()

	v := url.Values{}
	v.Set("include_entities", "false")
	v.Set("trim_user", "false")
	v.Set("map", "false")

	tweets, err := c.api.GetTweetsLookupByIds(ids, v)
	if err != nil {
		log.Error(err)
		return
	}

	var validIDs []int64
	for _, t := range tweets {
		validIDs = append(validIDs, t.Id)
	}

	validIDsCh <- validIDs
	log.Infof("Done %d", goroutineIndex)
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

	var ids []int64
	for {
		tweets, err := c.api.GetUserTimeline(v)
		if err != nil {
			return err
		} else if len(tweets) == 0 {
			return c.eraseIDs(ids)
		}

		for _, t := range tweets {
			ids = append(ids, t.Id)
		}

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
