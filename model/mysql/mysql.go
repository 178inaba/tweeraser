package mysql

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/go-sql-driver/mysql"
)

type beginner interface {
	Begin() (*sql.Tx, error)
}

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

type prepareRunner struct {
	preparer sq.Preparer
}

func (r prepareRunner) Query(query string, args ...interface{}) (*sql.Rows, error) {
	stmt, err := r.preparer.Prepare(query)
	if err != nil {
		return nil, err
	}

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (r prepareRunner) Exec(query string, args ...interface{}) (sql.Result, error) {
	stmt, err := r.preparer.Prepare(query)
	if err != nil {
		return nil, err
	}

	res, err := stmt.Exec(args...)
	if err != nil {
		return nil, err
	}

	return res, err
}

func (r prepareRunner) QueryRow(query string, args ...interface{}) sq.RowScanner {
	stmt, err := r.preparer.Prepare(query)
	if err != nil {
		return &row{err: err}
	}

	return &row{RowScanner: stmt.QueryRow(args...)}
}

type row struct {
	sq.RowScanner
	err error
}

func (r *row) Scan(dest ...interface{}) error {
	if r.err != nil {
		return r.err
	}

	return r.RowScanner.Scan(dest...)
}
