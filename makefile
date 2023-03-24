SHELL := /bin/bash

export PROJECT = ardan-starter-kit

# ==============================================================================
# Testing running system

# curl --user "admin@example.com:gophers" http://localhost:3000/v1/users/token
# export TOKEN="COPY TOKEN STRING FROM LAST CALL"
# curl -H "Authorization: Bearer ${TOKEN}" http://localhost:3000/v1/users
# hey -m GET -c 100 -n 10000 -H "Authorization: Bearer ${TOKEN}" http://localhost:3000/v1/users


# Expvarmon in the makefile
# GitHub - rakyll/hey: HTTP load generator, ApacheBench (ab) replacement
# go install github.com/divan/expvarmon@latest
# go install github.com/rakyll/hey@latest
# hey -m GET -c 100 -n 1000 http://localhost:3000/readiness/10

# ==============================================================================
# Building containers

all: sales metrics

sales:
	docker build \
		-f zarf/compose/dockerfile.sales-api \
		-t sales-api-amd64:1.0 \
		--build-arg PACKAGE_NAME=sales-api \
		--build-arg VCS_REF=`git rev-parse HEAD` \
		--build-arg BUILD_DATE=`date -u +”%Y-%m-%dT%H:%M:%SZ”` \
		.

metrics:
	docker build \
		-f zarf/compose/dockerfile.metrics \
		-t metrics-amd64:1.0 \
		--build-arg PACKAGE_NAME=metrics \
		--build-arg PACKAGE_PREFIX=sidecar/ \
		--build-arg VCS_REF=`git rev-parse HEAD` \
		--build-arg BUILD_DATE=`date -u +”%Y-%m-%dT%H:%M:%SZ”` \
		.

# ==============================================================================
# Running from within docker compose

run: up seed

up:
	docker-compose -f zarf/compose/compose.yaml -f zarf/compose/compose-config.yaml up 

down:
	docker-compose -f zarf/compose/compose.yaml down --remove-orphans

logs:
	docker-compose -f zarf/compose/compose.yaml logs -f

service-log:
	docker-compose -f zarf/compose/compose.yaml logs -f sales-api

# ==============================================================================
# Administration

migrate:
	go run app/sales-admin/main.go --db-disable-tls=1 migrate

seed: migrate
	go run app/sales-admin/main.go --db-disable-tls=1 seed

# ==============================================================================
# Running tests within the local computer

test:
	go test ./... -count=1
	staticcheck ./...

# ==============================================================================
# Modules support

deps-reset:
	git checkout -- go.mod
	go mod tidy
	go mod vendor

tidy:
	go mod tidy
	go mod vendor

deps-upgrade:
	# go get $(go list -f '{{if not (or .Main .Indirect)}}{{.Path}}{{end}}' -m all)
	go get -u -t -d -v ./...
	go mod tidy
	go mod vendor

deps-cleancache:
	go clean -modcache

# ==============================================================================
# Docker support

FILES := $(shell docker ps -aq)

down-local:
	docker stop $(FILES)
	docker rm $(FILES)

clean:
	docker system prune -f	

logs-local:
	docker logs -f $(FILES)

# ==============================================================================
