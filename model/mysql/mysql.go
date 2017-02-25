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
	canClose bool
}

func newPrepareRunner(preparer sq.Preparer) prepareRunner {
	canClose := false
	if _, ok := preparer.(*sql.DB); ok {
		canClose = true
	}

	return prepareRunner{preparer: preparer, canClose: canClose}
}

func (r prepareRunner) Query(query string, args ...interface{}) (*sql.Rows, error) {
	stmt, err := r.preparer.Prepare(query)
	if err != nil {
		return nil, err
	} else if r.canClose {
		defer stmt.Close()
	}

	return stmt.Query(args...)
}

func (r prepareRunner) Exec(query string, args ...interface{}) (sql.Result, error) {
	stmt, err := r.preparer.Prepare(query)
	if err != nil {
		return nil, err
	} else if r.canClose {
		defer stmt.Close()
	}

	return stmt.Exec(args...)
}

func (r prepareRunner) QueryRow(query string, args ...interface{}) sq.RowScanner {
	stmt, err := r.preparer.Prepare(query)
	if err != nil {
		return &row{err: err}
	} else if r.canClose {
		defer stmt.Close()
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

// SetMaxOpenConnsFromDB set max open connections from database configuration.
// Can specify the percentage.
// If 0 is specified for percentage, 0 is set.
func SetMaxOpenConnsFromDB(db *sql.DB, percentage int) error {
	// Get database max conns.
	var maxConns int
	err := db.QueryRow("SELECT @@max_connections").Scan(&maxConns)
	if err != nil {
		return err
	}

	// Set.
	db.SetMaxOpenConns(maxConns * percentage / 100)
	return nil
}
