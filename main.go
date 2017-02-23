package main

import (
	"archive/zip"
	"database/sql"
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
)

func main() {
	kingpin.Parse()
	os.Exit(run())
}

func run() int {
	c, err := newTweetEraseClient()
	if err != nil {
		log.Error(err)
		return 1
	}

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

func newTweetEraseClient() (*tweetEraseClient, error) {
	api, err := newAPI(configFilePath)
	if err != nil {
		return nil, err
	}

	var ets model.EraseTweetService
	var ees model.EraseErrorService
	db, err := newDB()
	if err == nil {
		ets = mysql.NewEraseTweetService(db)
		ees = mysql.NewEraseErrorService(db)
	} else {
		log.Warn(err)
	}

	// Create twitter user.
	tu, err := model.NewTwitterUser(api)
	if err != nil {
		return nil, err
	}

	// Insert twitter user.
	err = mysql.NewTwitterUserService(db).Insert(tu)
	if err != nil {
		return nil, err
	}

	return &tweetEraseClient{
		api: api, user: tu, eraseTweetService: ets, eraseErrorService: ees}, nil
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

func newDB() (*sql.DB, error) {
	db, err := mysql.Open("root", "", "tweeraser")
	if err != nil {
		return nil, errors.Errorf("Fail db open: %s.", err)
	}

	if err := db.Ping(); err != nil {
		return nil, errors.Errorf("Fail db ping: %s.", err)
	}

	return db, nil
}

type tweetEraseClient struct {
	api               *anaconda.TwitterApi
	user              *model.TwitterUser
	eraseTweetService model.EraseTweetService
	eraseErrorService model.EraseErrorService
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

	var ids []uint64
	for {
		record, err := cr.Read()
		if err == io.EOF {
			return c.checkBeforeEraseIDs(ids)
		} else if err != nil {
			return err
		}

		id, err := strconv.ParseInt(record[tweetIDIndex], 10, 64)
		if err != nil {
			return err
		}

		ids = append(ids, uint64(id))
	}
}

func (c tweetEraseClient) checkBeforeEraseIDs(ids []uint64) error {
	idsMap := map[uint64]struct{}{}
	for _, id := range ids {
		idsMap[id] = struct{}{}
	}

	inCount := 1000
	for inCount > 0 {
		tweetIDs, err := c.eraseTweetService.AlreadyEraseTweetIDs(c.user.UserID, ids[:inCount])
		if err != nil {
			return err
		}

		for _, id := range tweetIDs {
			delete(idsMap, id)
		}

		notFoundIDs, err := c.eraseErrorService.TweetNotFoundIDs(ids[:inCount])
		if err != nil {
			return err
		}

		for _, id := range notFoundIDs {
			delete(idsMap, id)
		}

		ids = append(ids[:0], ids[inCount:]...)
		idsLen := len(ids)
		if idsLen < inCount {
			inCount = idsLen
		}
	}

	validIDs := make([]uint64, 0, len(idsMap))
	for id := range idsMap {
		validIDs = append(validIDs, id)
	}

	return c.eraseIDs(validIDs)
}

func (c tweetEraseClient) eraseTimeline() error {
	v := url.Values{}
	v.Set("user_id", fmt.Sprint(c.user.UserID))
	v.Set("count", fmt.Sprint(200))
	v.Set("trim_user", "true")
	v.Set("contributor_details", "false")
	v.Set("include_rts", "false")

	var ids []uint64
	for {
		tweets, err := c.api.GetUserTimeline(v)
		if err != nil {
			return err
		} else if len(tweets) == 0 {
			return c.eraseIDs(ids)
		}

		for _, t := range tweets {
			ids = append(ids, uint64(t.Id))
		}

		v.Set("max_id", fmt.Sprint(tweets[len(tweets)-1].Id-1))
	}
}

func (c tweetEraseClient) eraseIDs(ids []uint64) error {
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

func (c tweetEraseClient) eraseTweet(id uint64, wg *sync.WaitGroup, isErrCh chan<- bool) {
	defer wg.Done()

	l := log.WithField("id", id)

	t, err := c.api.DeleteTweet(int64(id), true)
	if err != nil {
		insertID, insertErr := c.insertEraseError(id, err)
		if insertID != 0 && insertErr == nil {
			l = l.WithField("insert_id", insertID)
		} else if insertErr != nil {
			l.Errorf("Fail erase error insert: %s", insertErr)
		}

		l.Errorf("Fail erase: %s", err)
		isErrCh <- true
		return
	}

	insertID, err := c.insertEraseTweet(t)
	if err != nil {
		l.Errorf("Fail insert: %s", err)
		isErrCh <- true
		return
	} else if insertID != 0 {
		l = l.WithField("insert_id", insertID)
	}

	l.Info("Successfully erased!")
	isErrCh <- false
}

func (c tweetEraseClient) insertEraseTweet(t anaconda.Tweet) (uint64, error) {
	if c.eraseTweetService == nil {
		return 0, nil
	}

	postedAt, err := t.CreatedAtTime()
	if err != nil {
		return 0, err
	}

	et := &model.EraseTweet{
		TwitterTweetID: uint64(t.Id), Tweet: t.Text, PostedAt: postedAt}
	insertID, err := c.eraseTweetService.Insert(et)
	if err != nil {
		return 0, err
	}

	return insertID, nil
}

func (c tweetEraseClient) insertEraseError(tweetID uint64, err error) (uint64, error) {
	if c.eraseErrorService == nil {
		return 0, nil
	}

	var statusCode uint16
	if apiErr, ok := err.(*anaconda.ApiError); ok {
		statusCode = uint16(apiErr.StatusCode)
	}

	ee := &model.EraseError{TwitterTweetID: tweetID, StatusCode: statusCode, ErrorMessage: err.Error()}
	insertID, err := c.eraseErrorService.Insert(ee)
	if err != nil {
		return 0, err
	}

	return insertID, nil
}
