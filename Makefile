SHELL := /bin/sh
OUT := $(shell pwd)/_out
BUILDARCH := $(shell uname -m)
GCC := $(OUT)/$(BUILDARCH)-linux-musl-cross/bin/$(BUILDARCH)-linux-musl-gcc
LD := $(OUT)/$(BUILDARCH)-linux-musl-cross/bin/$(BUILDARCH)-linux-musl-ld

build: musl
	CGO_ENABLED=1 \
	CC_FOR_TARGET=$(GCC) \
	CC=$(GCC) \
	go build \
		-buildmode=c-shared \
		-ldflags '-linkmode external -extldflags -static' \
		-a -o "$(OUT)/lib.so" .

up:
	docker compose up

run: build up clean

musl:
	if [ ! -d "$(OUT)/$(BUILDARCH)-linux-musl-cross" ]; then \
		(cd $(OUT); curl -LOk https://musl.cc/$(BUILDARCH)-linux-musl-cross.tgz) && \
		tar zxf $(OUT)/$(BUILDARCH)-linux-musl-cross.tgz -C $(OUT); \
	fi

clean:
	docker compose rm -fsv
	sudo rm -Rf $(OUT) $(BINARY_NAME)
	mkdir -p $(OUT)
	touch $(OUT)/.keep
