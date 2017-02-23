package mysql

import (
	"database/sql"
	"time"

	"github.com/178inaba/tweeraser/model"
	sq "github.com/Masterminds/squirrel"
)

// TwitterUserService is twitter user table service.
type TwitterUserService struct {
	pr prepareRunner
}

// NewTwitterUserService is create twitter user service.
func NewTwitterUserService(db *sql.DB) TwitterUserService {
	return TwitterUserService{pr: prepareRunner{db: db}}
}

// Insert is insert to twitter user table.
func (s TwitterUserService) Insert(tu *model.TwitterUser) error {
	now := time.Now().UTC()
	query, args, err := sq.Insert(model.TwitterUserTableName).Columns(
		"user_id", "screen_name", "name", "lang", "updated_at", "created_at").
		Values(tu.UserID, tu.ScreenName, tu.Name, tu.Lang, now, now).ToSql()
	if err != nil {
		return err
	}

	_, err = s.pr.Exec(query, args...)
	if err != nil {
		return err
	}

	return nil
}
