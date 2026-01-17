module github.com/structx/pgxadpater/fxmigrate

go 1.25.5

require (
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/structx/pgxadapter v0.0.0-00010101000000-000000000000
	github.com/structx/pgxadapter/migrate v0.0.0-00010101000000-000000000000
	go.uber.org/fx v1.24.0
)

require (
	github.com/jackc/pgerrcode v0.0.0-20220416144525-469b46aa5efa // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.8.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.26.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.33.0 // indirect
)

replace (
	github.com/structx/pgxadapter => ../
	github.com/structx/pgxadapter/migrate => ../migrate
)
