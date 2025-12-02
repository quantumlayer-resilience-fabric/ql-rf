module github.com/quantumlayerhq/ql-rf/services/api

go 1.23

require (
	github.com/go-chi/chi/v5 v5.1.0
	github.com/go-chi/cors v1.2.1
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.7.1
	github.com/quantumlayerhq/ql-rf v0.0.0
)

replace github.com/quantumlayerhq/ql-rf => ../..
