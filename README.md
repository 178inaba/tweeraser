# Tweeraser

[![Build Status](https://travis-ci.org/178inaba/tweeraser.svg?branch=master)](https://travis-ci.org/178inaba/tweeraser)
[![Coverage Status](https://coveralls.io/repos/github/178inaba/tweeraser/badge.svg?branch=master)](https://coveralls.io/github/178inaba/tweeraser?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/178inaba/tweeraser)](https://goreportcard.com/report/github.com/178inaba/tweeraser)

Tweeraser will erase all tweets.

## Test

Require MySQL or MariaDB.

```console
$ mysql -u root < misc/sql/create_test_db.sql
$ mysql -u root tweeraser_test < misc/sql/ddl.sql
$ go test ./...
```
