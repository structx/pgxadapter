module github.com/structx/pgxadapter/migrate

go 1.25.5

require (
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/jackc/pgerrcode v0.0.0-20220416144525-469b46aa5efa
	github.com/jackc/pgx/v5 v5.8.0
	github.com/structx/pgxadapter v0.0.0-00010101000000-000000000000
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/text v0.33.0 // indirect
)

replace github.com/structx/pgxadapter => ../
