package mysql

import (
	"database/sql"
	"time"

	"github.com/178inaba/tweeraser/model"
	sq "github.com/Masterminds/squirrel"
)

// EraseTweetService is mysql database service.
type EraseTweetService struct {
	pr prepareRunner
}

// NewEraseTweetService is create service.
func NewEraseTweetService(db *sql.DB) EraseTweetService {
	return EraseTweetService{pr: prepareRunner{preparer: db}}
}

// AlreadyEraseTweetIDs return already erase ids from argument ids.
func (s EraseTweetService) AlreadyEraseTweetIDs(userID uint64, ids []uint64) ([]uint64, error) {
	query, args, err := sq.Select("twitter_tweet_id").From(model.EraseTweetTableName).
		Where(sq.Eq{"twitter_user_id": userID, "twitter_tweet_id": ids}).ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.pr.Query(query, args...)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tweetIDs []uint64
	for rows.Next() {
		var id uint64
		err := rows.Scan(&id)
		if err != nil {
			return nil, err
		}

		tweetIDs = append(tweetIDs, id)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return tweetIDs, nil
}

// Insert is insert erase_tweets table.
func (s EraseTweetService) Insert(et *model.EraseTweet) (uint64, error) {
	now := time.Now().UTC()
	query, args, err := sq.Insert(model.EraseTweetTableName).Columns(
		"twitter_tweet_id", "tweet", "posted_at", "twitter_user_id", "updated_at", "created_at").
		Values(et.TwitterTweetID, et.Tweet, et.PostedAt, et.TwitterUserID, now, now).ToSql()
	if err != nil {
		return 0, err
	}

	res, err := s.pr.Exec(query, args...)
	if err != nil {
		return 0, err
	}

	lastInsertID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return uint64(lastInsertID), nil
}
