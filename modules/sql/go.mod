module github.com/itrn0/risor/modules/sql

go 1.22.0

toolchain go1.23.1

replace github.com/itrn0/risor => ../..

require (
	github.com/go-sql-driver/mysql v1.8.1
	github.com/lib/pq v1.10.9
	github.com/mattn/go-sqlite3 v1.14.24
	github.com/microsoft/go-mssqldb v1.7.2
	github.com/itrn0/risor v1.7.0
	github.com/xo/dburl v0.23.2
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	golang.org/x/crypto v0.26.0 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/text v0.17.0 // indirect
)
