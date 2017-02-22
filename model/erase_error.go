package model

import "time"

// EraseErrorTableName is erase error table name.
const EraseErrorTableName = "erase_errors"

// EraseError is erace error object.
type EraseError struct {
	ID             uint64
	TwitterTweetID uint64
	StatusCode     uint16
	UpdatedAt      time.Time
	CreatedAt      time.Time
}

// EraseErrorService is erase error service interface.
type EraseErrorService interface {
	EraseErrorTweetIDs(ids []uint64) ([]uint64, error)
	Insert(ee *EraseError) (uint64, error)
}
