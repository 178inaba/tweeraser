package model

import (
	"net/url"
	"time"

	"github.com/ChimeraCoder/anaconda"
)

// TwitterUserTableName is twitter user table name.
const TwitterUserTableName = "twitter_users"

// TwitterUser is twitter user object.
type TwitterUser struct {
	UserID     uint64
	ScreenName string
	Name       string
	Lang       string
	UpdatedAt  time.Time
	CreatedAt  time.Time
}

// NewTwitterUser create TwitterUser from twitter api.
func NewTwitterUser(api *anaconda.TwitterApi) (*TwitterUser, error) {
	v := url.Values{}
	v.Set("include_entities", "false")
	v.Set("skip_status", "true")
	v.Set("include_email", "false")

	u, err := api.GetSelf(v)
	if err != nil {
		return nil, err
	}

	return &TwitterUser{UserID: uint64(u.Id),
		ScreenName: u.ScreenName, Name: u.Name, Lang: u.Lang}, nil
}

// TwitterUserService is twitter user service interface.
type TwitterUserService interface {
	Insert(tu *TwitterUser) error
}
