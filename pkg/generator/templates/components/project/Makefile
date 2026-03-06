.PHONY: help install build run test clean docker

help:
	@echo "{{.ProjectName}} - Go API 项目"

install:
	go mod download
	go mod tidy

build:
	go build -o bin/app ./cmd/

run: install
	go run ./cmd/main.go

test:
	go test ./... -v

clean:
	rm -rf bin/
	go clean

docker:
	docker-compose up
