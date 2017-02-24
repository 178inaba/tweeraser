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

// CheckWantUpdate checks if it needs updating compared with the argument object.
// Target data to be updated if there is a difference: ScreenName, Name, Lang.
func (t *TwitterUser) CheckWantUpdate(tu *TwitterUser) bool {
	if t.ScreenName != tu.ScreenName {
		return true
	} else if t.Name != tu.Name {
		return true
	} else if t.Lang != tu.Lang {
		return true
	}

	return false
}

// TwitterUserService is twitter user service interface.
type TwitterUserService interface {
	InsertUpdate(tu *TwitterUser) error
}
