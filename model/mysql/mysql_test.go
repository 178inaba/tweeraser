package mysql_test

import (
	"testing"

	"github.com/178inaba/tweeraser/model/mysql"
	"github.com/stretchr/testify/assert"
)

func TestSetMaxOpenConnsFromDB(t *testing.T) {
	db, err := mysql.Open("root", "", "tweeraser_test")
	assert.NoError(t, err)
	defer db.Close()

	err = mysql.SetMaxOpenConnsFromDB(db, 90)
	assert.NoError(t, err)
}
