package mysql_test

import (
	"database/sql"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/178inaba/tweeraser/model"
	"github.com/178inaba/tweeraser/model/mysql"
	sq "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/suite"
)

type twitterUserSuite struct {
	suite.Suite

	db      *sql.DB
	service model.TwitterUserService
}

func TestTwitterUserSuite(t *testing.T) {
	suite.Run(t, new(twitterUserSuite))
}

func (s *twitterUserSuite) SetupSuite() {
	db, err := mysql.Open("root", "", "tweeraser_test")
	s.NoError(err)

	s.db = db
	s.service = mysql.NewTwitterUserService(db)
}

func (s *twitterUserSuite) SetupTest() {
	// Reset test db.
	_, err := s.db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	s.NoError(err)
	_, err = s.db.Exec(fmt.Sprintf("TRUNCATE TABLE %s", model.TwitterUserTableName))
	s.NoError(err)
	_, err = s.db.Exec("SET FOREIGN_KEY_CHECKS = 1")
	s.NoError(err)
}

func (s *twitterUserSuite) TestInsertUpdate() {
	tu := &model.TwitterUser{UserID: math.MaxUint64,
		ScreenName: "screen_name", Name: "name", Lang: "en"}
	err := s.service.InsertUpdate(tu)
	s.NoError(err)

	rows, err := sq.Select("*").
		From(model.TwitterUserTableName).RunWith(s.db).Query()
	s.NoError(err)

	var cnt int
	for rows.Next() {
		var actual model.TwitterUser
		err := rows.Scan(&actual.UserID, &actual.ScreenName,
			&actual.Name, &actual.Lang, &actual.UpdatedAt, &actual.CreatedAt)
		s.NoError(err)

		s.Equal(tu.UserID, actual.UserID)
		s.Equal(tu.ScreenName, actual.ScreenName)
		s.Equal(tu.Name, actual.Name)
		s.Equal(tu.Lang, actual.Lang)

		threeSecAgo := time.Now().UTC().Add(-3 * time.Second)
		s.True(actual.UpdatedAt.After(threeSecAgo))
		s.True(actual.CreatedAt.After(threeSecAgo))

		cnt++
	}

	s.Equal(1, cnt)
	s.NoError(rows.Err())

	// Duplicate update.
	tu = &model.TwitterUser{UserID: math.MaxUint64, Name: "name_dup"}
	err = s.service.InsertUpdate(tu)
	s.NoError(err)

	rows, err = sq.Select("name").
		From(model.TwitterUserTableName).RunWith(s.db).Query()
	s.NoError(err)

	cnt = 0
	for rows.Next() {
		var actual model.TwitterUser
		err := rows.Scan(&actual.Name)
		s.NoError(err)
		s.Equal(tu.Name, actual.Name)
		cnt++
	}

	s.Equal(1, cnt)
	s.NoError(rows.Err())
}

func (s *twitterUserSuite) TearDownSuite() {
	s.db.Close()
}
