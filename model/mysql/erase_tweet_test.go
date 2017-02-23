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
	_, err = s.db.Exec(fmt.Sprintf("TRUNCATE TABLE %s", model.TwitterUserTableName))
	s.NoError(err)
	_, err = s.db.Exec("SET FOREIGN_KEY_CHECKS = 1")
	s.NoError(err)

	// Create test twitter users.
	tus := mysql.NewTwitterUserService(s.db)
	for _, uid := range []uint64{1, 2, math.MaxUint64} {
		tu := &model.TwitterUser{UserID: uid}
		err = tus.Insert(tu)
		s.NoError(err)
	}
}

func (s *eraseTweetTestSuite) TestAlreadyEraseTweetIDs() {
	userID := uint64(1)
	cnt := 1000
	ids := make([]uint64, cnt)
	dummyIDs := make([]uint64, cnt)
	for i := 1; i <= cnt; i++ {
		dummyID := math.MaxUint64 - uint64(i)
		ids[i-1] = dummyID
		dummyIDs[i-1] = dummyID
		et := &model.EraseTweet{TwitterTweetID: dummyID, TwitterUserID: userID}
		insertID, err := s.service.Insert(et)
		s.NoError(err)
		s.Equal(uint64(i), insertID)
	}

	// Other user.
	et := &model.EraseTweet{TwitterTweetID: 10000, TwitterUserID: 2}
	insertID, err := s.service.Insert(et)
	s.NoError(err)
	s.Equal(uint64(cnt+1), insertID)

	ids = append(ids, []uint64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}...)
	tweetIDs, err := s.service.AlreadyEraseTweetIDs(userID, ids)
	s.NoError(err)
	s.Len(tweetIDs, cnt)

	for _, dummyID := range dummyIDs {
		var isExist bool
		for _, tweetID := range tweetIDs {
			if tweetID == dummyID {
				isExist = true
				break
			}
		}

		s.True(isExist)
	}
}

func (s *eraseTweetTestSuite) TestInsert() {
	tweet140 := "12345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890"
	postedAt := time.Now().Add(-24 * time.Hour).UTC()
	et := &model.EraseTweet{TwitterTweetID: math.MaxUint64,
		Tweet: tweet140, PostedAt: postedAt, TwitterUserID: math.MaxUint64}
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
			&actual.PostedAt, &actual.TwitterUserID, &actual.UpdatedAt, &actual.CreatedAt)
		s.NoError(err)

		s.Equal(insertID, actual.ID)
		s.Equal(et.TwitterTweetID, actual.TwitterTweetID)
		s.Equal(et.Tweet, actual.Tweet)
		s.WithinDuration(et.PostedAt.Truncate(time.Second), actual.PostedAt, 0)
		s.Equal(et.TwitterUserID, actual.TwitterUserID)

		threeSecAgo := time.Now().UTC().Add(-3 * time.Second)
		s.True(actual.UpdatedAt.After(threeSecAgo))
		s.True(actual.CreatedAt.After(threeSecAgo))

		cnt++
	}

	s.Equal(1, cnt)
	s.NoError(rows.Err())

	// Not exist user.
	insertID, err = s.service.Insert(&model.EraseTweet{TwitterUserID: 3})
	s.Error(err)
	s.Equal(uint64(0), insertID)
}

func (s *eraseTweetTestSuite) TearDownSuite() {
	s.db.Close()
}
