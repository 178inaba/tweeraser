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
	defer c.close()

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
	conf, err := config.LoadConfig(configFilePath)
	if err != nil {
		return nil, err
	}

	api, err := newAPI(conf)
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
	err = mysql.NewTwitterUserService(db).InsertUpdate(tu)
	if err != nil {
		return nil, err
	}

	return &tweetEraseClient{
		config: conf, api: api, user: tu, db: db, eraseTweetService: ets, eraseErrorService: ees}, nil
}

func newAPI(conf *config.Config) (*anaconda.TwitterApi, error) {
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

	if err := mysql.SetMaxOpenConnsFromDB(db, 60); err != nil {
		return nil, err
	}

	return db, nil
}

type tweetEraseClient struct {
	config            *config.Config
	api               *anaconda.TwitterApi
	user              *model.TwitterUser
	db                *sql.DB
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

		notFoundIDs, err := c.eraseErrorService.TweetNotFoundIDs(c.user.UserID, ids[:inCount])
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
	v.Set("exclude_replies", "false")
	v.Set("contributor_details", "false")
	v.Set("include_rts", "true")

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
	trialCnt := 1000
	idsLen := len(ids)
	if idsLen < 1000 {
		trialCnt = idsLen
	}

	for trialCnt > 0 {
		wg := new(sync.WaitGroup)
		for _, id := range ids[:trialCnt] {
			wg.Add(1)
			go c.eraseTweet(id, wg)
		}

		wg.Wait()

		ids = append(ids[:0], ids[trialCnt:]...)
		idsLen := len(ids)
		if idsLen < trialCnt {
			trialCnt = idsLen
		}
	}

	return nil
}

func (c tweetEraseClient) eraseTweet(id uint64, wg *sync.WaitGroup) {
	defer wg.Done()

	l := log.WithField("id", id)

	// Create api.
	api, err := newAPI(c.config)
	if err != nil {
		l.Errorf("Fail create api: %s", err)
		return
	}
	defer api.Close()

	t, err := api.DeleteTweet(int64(id), true)
	if err != nil {
		insertID, insertErr := c.insertEraseError(id, err)
		if insertID != 0 && insertErr == nil {
			l = l.WithField("insert_id", insertID)
		} else if insertErr != nil {
			l.Errorf("Fail erase error insert: %s", insertErr)
		}

		l.Errorf("Fail erase: %s", err)
		return
	}

	insertID, err := c.insertEraseTweet(t)
	if err != nil {
		l.Errorf("Fail erase tweet insert: %s", err)
		return
	} else if insertID != 0 {
		postedAt, err := t.CreatedAtTime()
		if err != nil {
			l.Errorf("Fail parse posted at: %s", err)
			return
		}

		l = l.WithFields(log.Fields{"insert_id": insertID,
			"tweet": t.Text, "posted_at": postedAt.Format("2006-01-02 15:04:05")})
	}

	l.Info("Successfully erased!")
}

func (c tweetEraseClient) insertEraseTweet(t anaconda.Tweet) (uint64, error) {
	if c.eraseTweetService == nil {
		return 0, nil
	}

	postedAt, err := t.CreatedAtTime()
	if err != nil {
		return 0, err
	}

	et := &model.EraseTweet{TwitterTweetID: uint64(t.Id),
		Tweet: t.Text, PostedAt: postedAt, TwitterUserID: uint64(t.User.Id)}
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

	ee := &model.EraseError{TriedTwitterUserID: c.user.UserID, TwitterTweetID: tweetID, StatusCode: statusCode, ErrorMessage: err.Error()}
	insertID, err := c.eraseErrorService.Insert(ee)
	if err != nil {
		return 0, err
	}

	return insertID, nil
}

func (c tweetEraseClient) close() error {
	c.api.Close()

	if err := c.db.Close(); err != nil {
		return err
	}

	return nil
}
