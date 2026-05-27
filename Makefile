SCRIPTS_DIR ?= $(HOME)/Development/github.com/rios0rios0/pipelines
-include $(SCRIPTS_DIR)/makefiles/common.mk
-include $(SCRIPTS_DIR)/makefiles/golang.mk

VERSION ?= $(shell { git describe --tags --abbrev=0 2>/dev/null || echo "dev"; } | sed 's/^v//')
LDFLAGS := -X main.version=$(VERSION)

.PHONY: debug build install run release

build:
	rm -rf bin
	go build -ldflags "$(LDFLAGS) -s -w" -o bin/chlog .

debug:
	rm -rf bin
	go build -gcflags "-N -l" -ldflags "$(LDFLAGS)" -o bin/chlog .

run:
	go run .

install:
	make build
	mkdir -p ~/.local/bin
	cp -v bin/chlog ~/.local/bin/chlog

release: build
	bin/chlog batch auto
	bin/chlog merge
