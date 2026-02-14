module github.com/max-cloud/api

go 1.25.7

require (
	github.com/go-chi/chi/v5 v5.2.5
	github.com/google/uuid v1.6.0
	github.com/max-cloud/shared v0.0.0
)

replace github.com/max-cloud/shared => ../../packages/shared
