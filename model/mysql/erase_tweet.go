package mysql

import (
	"database/sql"
	"time"

	"github.com/178inaba/tweeraser/model"
	sq "github.com/Masterminds/squirrel"
)

// EraseTweetService is mysql database service.
type EraseTweetService struct {
	pe prepareExecer
}

// NewEraseTweetService is create service.
func NewEraseTweetService(db *sql.DB) EraseTweetService {
	return EraseTweetService{pe: prepareExecer{db: db}}
}

// Insert is insert erase_tweets table.
func (s EraseTweetService) Insert(et *model.EraseTweet) (uint64, error) {
	now := time.Now().UTC()
	sql, args, err := sq.Insert(model.EraseTweetTableName).Columns(
		"twitter_tweet_id", "tweet", "posted_at", "updated_at", "created_at").
		Values(et.TwitterTweetID, et.Tweet, et.PostedAt, now, now).ToSql()
	if err != nil {
		return 0, err
	}

	res, err := s.pe.Exec(sql, args...)
	if err != nil {
		return 0, err
	}

	lastInsertID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return uint64(lastInsertID), nil
}
