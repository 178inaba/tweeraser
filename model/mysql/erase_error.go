package mysql

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/178inaba/tweeraser/model"
	sq "github.com/Masterminds/squirrel"
)

// EraseErrorService is erase errors table service.
type EraseErrorService struct {
	pr prepareRunner
}

// NewEraseErrorService is create erase error service.
func NewEraseErrorService(db *sql.DB) EraseErrorService {
	return EraseErrorService{pr: newPrepareRunner(db)}
}

// TweetNotFoundIDs return not found tweet ids from argument ids.
func (s EraseErrorService) TweetNotFoundIDs(userID uint64, ids []uint64) ([]uint64, error) {
	query, args, err := sq.Select("twitter_tweet_id").From(model.EraseErrorTableName).
		Where(sq.Eq{"tried_twitter_user_id": userID,
			"status_code": http.StatusNotFound, "twitter_tweet_id": ids}).ToSql()
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

// Insert is insert to erase error table.
func (s EraseErrorService) Insert(ee *model.EraseError) (uint64, error) {
	now := time.Now().UTC()
	query, args, err := sq.Insert(model.EraseErrorTableName).Columns(
		"tried_twitter_user_id", "twitter_tweet_id",
		"status_code", "error_message", "updated_at", "created_at").
		Values(ee.TriedTwitterUserID, ee.TwitterTweetID, ee.StatusCode, ee.ErrorMessage, now, now).ToSql()
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
