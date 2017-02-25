package mysql

import (
	"database/sql"
	"time"

	"github.com/178inaba/tweeraser/model"
	sq "github.com/Masterminds/squirrel"
	"github.com/pkg/errors"
)

// TwitterUserService is twitter user table service.
type TwitterUserService struct {
	preparer sq.Preparer
	pr       prepareRunner
}

// NewTwitterUserService is create twitter user service.
// When calling InsertUpdate, specify an object implementing `Begin() (*sql.Tx, error)` (e.g. *sql.DB) as an argument.
func NewTwitterUserService(preparer sq.Preparer) TwitterUserService {
	return TwitterUserService{preparer: preparer, pr: newPrepareRunner(preparer)}
}

// InsertUpdate inserts if there is no line corresponding to the primary key, and updates if it does.
func (s TwitterUserService) InsertUpdate(tu *model.TwitterUser) (err error) {
	// Begin transaction.
	beginner, ok := s.preparer.(beginner)
	if !ok {
		return errors.New("preparer has no method Begin")
	}

	tx, err := beginner.Begin()
	if err != nil {
		return err
	}

	// Rollback.
	defer func() {
		if pv := recover(); pv != nil {
			switch v := pv.(type) {
			case error:
				err = v
			default:
				err = errors.Errorf("%s", v)
			}
		}

		if err == nil {
			err = tx.Commit()
		} else if rErr := tx.Rollback(); rErr != nil {
			err = errors.Wrap(err, rErr.Error())
		}
	}()

	txService := NewTwitterUserService(tx)

	// Exist?
	dbtu, err := txService.selectForUpdate(tu.UserID)
	if err == sql.ErrNoRows {
		// Not Exist.
		// Insert.
		err := txService.insert(tu)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if tu.CheckWantUpdate(dbtu) { // Exist duplicate key object. Check want update.
		// Update.
		err := txService.update(tu)
		if err != nil {
			return err
		}
	}

	return nil
}

// insert is insert to twitter user table.
func (s TwitterUserService) insert(tu *model.TwitterUser) error {
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

// update is update to twitter user table.
func (s TwitterUserService) update(tu *model.TwitterUser) error {
	setMap := map[string]interface{}{"screen_name": tu.ScreenName,
		"name": tu.Name, "lang": tu.Lang, "updated_at": time.Now().UTC()}
	query, args, err := sq.Update(model.TwitterUserTableName).
		SetMap(setMap).Where(sq.Eq{"user_id": tu.UserID}).ToSql()
	if err != nil {
		return err
	}

	res, err := s.pr.Exec(query, args...)
	if err != nil {
		return err
	}

	updateCnt, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if updateCnt < 1 {
		return errors.Errorf("row not found: %d", tu.UserID)
	}

	return nil
}

func (s TwitterUserService) selectForUpdate(userID uint64) (*model.TwitterUser, error) {
	query, args, err := sq.Select("*").From(model.TwitterUserTableName).
		Where(sq.Eq{"user_id": userID}).Suffix("FOR UPDATE").ToSql()
	if err != nil {
		return nil, err
	}

	tu := &model.TwitterUser{}
	err = s.pr.QueryRow(query, args...).Scan(&tu.UserID,
		&tu.ScreenName, &tu.Name, &tu.Lang, &tu.UpdatedAt, &tu.CreatedAt)
	if err != nil {
		return nil, err
	}

	return tu, nil
}
