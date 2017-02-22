package mysql

import (
	"database/sql"
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
	return EraseErrorService{pr: prepareRunner{db: db}}
}

// EraseErrorTweetIDs return erase error tweet ids from argument ids.
func (s EraseErrorService) EraseErrorTweetIDs(ids []uint64) ([]uint64, error) {
	query, args, err := sq.Select("twitter_tweet_id").
		From(model.EraseErrorTableName).Where(sq.Eq{"twitter_tweet_id": ids}).ToSql()
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
		"twitter_tweet_id", "status_code", "updated_at", "created_at").
		Values(ee.TwitterTweetID, ee.StatusCode, now, now).ToSql()
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
