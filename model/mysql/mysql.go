package mysql

import (
	"database/sql"

	"github.com/go-sql-driver/mysql"
)

// Open is open mysql connection.
func Open(user, addr, dbName string) (*sql.DB, error) {
	c := &mysql.Config{
		User:      user,
		Net:       "tcp",
		Addr:      addr,
		DBName:    dbName,
		ParseTime: true,
	}

	return sql.Open("mysql", c.FormatDSN())
}

type prepareExecer struct {
	db *sql.DB
}

func (r prepareExecer) Exec(query string, args ...interface{}) (sql.Result, error) {
	stmt, err := r.db.Prepare(query)
	if err != nil {
		return nil, err
	}

	res, err := stmt.Exec(args...)
	if err != nil {
		return nil, err
	}

	return res, err
}
