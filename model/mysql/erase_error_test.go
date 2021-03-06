package mysql_test

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/178inaba/tweeraser/model"
	"github.com/178inaba/tweeraser/model/mysql"
	sq "github.com/Masterminds/squirrel"
	"github.com/stretchr/testify/suite"
)

type eraseErrorSuite struct {
	suite.Suite

	db      *sql.DB
	service model.EraseErrorService
}

func TestEraseErrorSuite(t *testing.T) {
	suite.Run(t, new(eraseErrorSuite))
}

func (s *eraseErrorSuite) SetupSuite() {
	db, err := mysql.Open("root", "", "tweeraser_test")
	s.NoError(err)

	s.db = db
	s.service = mysql.NewEraseErrorService(db)
}

func (s *eraseErrorSuite) SetupTest() {
	// Reset test db.
	_, err := s.db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	s.NoError(err)
	_, err = s.db.Exec(fmt.Sprintf("TRUNCATE TABLE %s", model.EraseErrorTableName))
	s.NoError(err)
	_, err = s.db.Exec(fmt.Sprintf("TRUNCATE TABLE %s", model.TwitterUserTableName))
	s.NoError(err)
	_, err = s.db.Exec("SET FOREIGN_KEY_CHECKS = 1")
	s.NoError(err)

	// Create test twitter users.
	tus := mysql.NewTwitterUserService(s.db)
	for _, uid := range []uint64{1, 2, math.MaxUint64} {
		tu := &model.TwitterUser{UserID: uid}
		err = tus.InsertUpdate(tu)
		s.NoError(err)
	}
}

func (s *eraseErrorSuite) TestTweetNotFoundIDs() {
	userID := uint64(1)
	cnt := 1000
	ids := make([]uint64, cnt)
	dummyIDs := make([]uint64, cnt)
	for i := 1; i <= cnt; i++ {
		dummyID := math.MaxUint64 - uint64(i)
		ids[i-1] = dummyID
		dummyIDs[i-1] = dummyID
		ee := &model.EraseError{TriedTwitterUserID: userID, TwitterTweetID: dummyID, StatusCode: http.StatusNotFound}
		insertID, err := s.service.Insert(ee)
		s.NoError(err)
		s.Equal(uint64(i), insertID)
	}

	// Other status.
	ee := &model.EraseError{TriedTwitterUserID: userID, TwitterTweetID: 100,
		StatusCode: http.StatusInternalServerError, ErrorMessage: "Error: status 500."}
	insertID, err := s.service.Insert(ee)
	s.NoError(err)
	s.Equal(uint64(cnt+1), insertID)

	// Other user.
	ee = &model.EraseError{TriedTwitterUserID: 2, TwitterTweetID: 100,
		StatusCode: http.StatusNotFound, ErrorMessage: "Error: status 404."}
	insertID, err = s.service.Insert(ee)
	s.NoError(err)
	s.Equal(uint64(cnt+2), insertID)

	ids = append(ids, []uint64{ee.TwitterTweetID, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}...)
	tweetIDs, err := s.service.TweetNotFoundIDs(userID, ids)
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

func (s *eraseErrorSuite) TestInsert() {
	ee := &model.EraseError{TriedTwitterUserID: math.MaxUint64, TwitterTweetID: math.MaxUint64,
		StatusCode: http.StatusNotFound, ErrorMessage: "Error: status 404."}
	insertID, err := s.service.Insert(ee)
	s.NoError(err)
	s.Equal(uint64(1), insertID)

	rows, err := sq.Select("*").
		From(model.EraseErrorTableName).RunWith(s.db).Query()
	s.NoError(err)

	var cnt int
	for rows.Next() {
		var actual model.EraseError
		err := rows.Scan(&actual.ID, &actual.TriedTwitterUserID, &actual.TwitterTweetID, &actual.StatusCode,
			&actual.ErrorMessage, &actual.UpdatedAt, &actual.CreatedAt)
		s.NoError(err)

		s.Equal(insertID, actual.ID)
		s.Equal(ee.TriedTwitterUserID, actual.TriedTwitterUserID)
		s.Equal(ee.TwitterTweetID, actual.TwitterTweetID)
		s.Equal(ee.StatusCode, actual.StatusCode)
		s.Equal(ee.ErrorMessage, actual.ErrorMessage)

		threeSecAgo := time.Now().UTC().Add(-3 * time.Second)
		s.True(actual.UpdatedAt.After(threeSecAgo))
		s.True(actual.CreatedAt.After(threeSecAgo))

		cnt++
	}

	s.Equal(1, cnt)
	s.NoError(rows.Err())
	s.NoError(rows.Close())
}

func (s *eraseErrorSuite) TearDownSuite() {
	s.db.Close()
}
