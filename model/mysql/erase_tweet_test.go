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

type eraseTweetTestSuite struct {
	suite.Suite

	db      *sql.DB
	service model.EraseTweetService
}

func TestEraseTweetSuite(t *testing.T) {
	suite.Run(t, new(eraseTweetTestSuite))
}

func (s *eraseTweetTestSuite) SetupSuite() {
	db, err := mysql.Open("root", "", "tweeraser_test")
	s.NoError(err)

	s.db = db
	s.service = mysql.NewEraseTweetService(db)
}

func (s *eraseTweetTestSuite) SetupTest() {
	// Reset test db.
	_, err := s.db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	s.NoError(err)
	_, err = s.db.Exec(fmt.Sprintf("TRUNCATE TABLE %s", model.EraseTweetTableName))
	s.NoError(err)
	_, err = s.db.Exec("SET FOREIGN_KEY_CHECKS = 1")
	s.NoError(err)
}

func (s *eraseTweetTestSuite) TestInsert() {
	tweet140 := "12345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890"
	postedAt := time.Now().Add(-24 * time.Hour).UTC()
	et := &model.EraseTweet{TwitterTweetID: math.MaxUint64, Tweet: tweet140, PostedAt: postedAt}
	insertID, err := s.service.Insert(et)
	s.NoError(err)
	s.Equal(uint64(1), insertID)

	rows, err := sq.Select("*").
		From(model.EraseTweetTableName).RunWith(s.db).Query()
	s.NoError(err)

	var cnt int
	for rows.Next() {
		var actual model.EraseTweet
		err := rows.Scan(&actual.ID, &actual.TwitterTweetID, &actual.Tweet,
			&actual.PostedAt, &actual.UpdatedAt, &actual.CreatedAt)
		s.NoError(err)

		s.Equal(insertID, actual.ID)
		s.Equal(et.TwitterTweetID, actual.TwitterTweetID)
		s.Equal(et.Tweet, actual.Tweet)
		s.Equal(et.PostedAt.Truncate(time.Second), actual.PostedAt)

		threeSecAgo := time.Now().UTC().Add(-3 * time.Second)
		s.True(actual.UpdatedAt.After(threeSecAgo))
		s.True(actual.CreatedAt.After(threeSecAgo))

		cnt++
	}

	s.Equal(1, cnt)
	s.NoError(rows.Err())
}

func (s *eraseTweetTestSuite) TearDownSuite() {
	s.db.Close()
}
