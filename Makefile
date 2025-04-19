.PHONY: build
build: gen
	go build -o td ./cmd/td.go

.PHONY: gen
gen:
	go install tool
	gqlclientgen -s schema.graphql -q queries.graphql -o internal/gql.go -n re

.PHONY: install
install:
	go install ./cmd/re

.PHONY: install-tools
install-tools:
	go install tool

.PHONY: lint
lint: install-tools
	staticcheck ./...
