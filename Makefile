SHELL := /bin/sh
OUT := $(shell pwd)/_out
BUILDARCH := $(shell uname -m)

run: build up clean

build:
	CGO_ENABLED=1 \
	go build \
		-buildmode=c-shared \
		-a -o "$(OUT)/lib.so" .

up:
	docker compose up

down:
	docker compose rm -fsv

clean: down
	sudo rm -Rf $(OUT) $(BINARY_NAME)
	mkdir -p $(OUT)
	touch $(OUT)/.keep
