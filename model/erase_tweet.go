package model

import "time"

// EraseTweetTableName is erase tweet table name.
const EraseTweetTableName = "erase_tweets"

// EraseTweet is erace tweet object.
type EraseTweet struct {
	ID             uint64
	TwitterTweetID uint64
	Tweet          string
	PostedAt       time.Time
	UpdatedAt      time.Time
	CreatedAt      time.Time
}

// EraseTweetService is service interface.
type EraseTweetService interface {
	ValidIDs(ids []uint64) ([]uint64, error)
	Insert(et *EraseTweet) (uint64, error)
}
