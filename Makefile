.PHONY: test image bin/ssm-env

IMAGE_NAME ?= ssm-env
VERSION ?= 0.0.2

bin/ssm-env: *.go
	go build -o $@ .

test:
	go test -race $(shell go list ./... | grep -v /vendor/)

image:
	docker build -t $(IMAGE_NAME):$(VERSION) -f docker/Dockerfile .
