export GOPATH:=$(shell pwd)

GO        ?= go
PKG       := ./src/sshalama/

# TODO: Do we also want to run with debug in production?
# the github.com/rjeczalik/notify prints a lot of debug
# stuff when this is set.
BUILDTAGS := debug
VERSION   ?= $(shell git describe --dirty --tags | sed 's/^v//' )

.PHONY: default
default: all

# find src/ -name .git -type d | sed -s 's/.git$//' | while read line; do echo -n "${line} " | sed 's/.\/src\///'; git -C $line rev-parse HEAD; done | sort > GLOCKFILE
.PHONY: deps
deps: bin/go-bindata
	go get -tags '$(BUILDTAGS)' -d -v sshalama/...
	go get github.com/robfig/glock
	git diff /dev/null GLOCKFILE | ./bin/glock apply .

.PHONY: sshalama
sshalama: deps binary

.PHONY: binary
binary: LDFLAGS += -X "main.buildTag=v$(VERSION)"
binary: LDFLAGS += -X "main.buildTime=$(shell date -u '+%Y-%m-%d %H:%M:%S UTC')"
binary: # assets
	go install -tags '$(BUILDTAGS)' -ldflags '$(LDFLAGS)' sshalama
	go install -tags '$(BUILDTAGS)' -ldflags '$(LDFLAGS)' sshalama/worker
#	go install -race -tags '$(BUILDTAGS)' -ldflags '$(LDFLAGS)' sshalama

.PHONY: release
release: BUILDTAGS=release
release: sshalama

.PHONY: bin/go-bindata
bin/go-bindata:
	GOOS="" GOARCH="" go get github.com/jteeuwen/go-bindata/go-bindata

assets: bin/go-bindata
	bin/go-bindata -nomemcopy -pkg=assets -prefix="assets/" -tags='$(BUILDTAGS)' \
                -debug=$(if $(findstring debug,$(BUILDTAGS)),true,false) \
                -o=src/sshalama/assets/assets_$(BUILDTAGS).go \
                assets/...

.PHONY: fmt
fmt:
	go fmt sshalama/...

.PHONY: all
all: fmt sshalama

.PHONY: clean
clean:
	rm -rf bin/
	rm -rf pkg/
	rm -rf src/sshalama/assets/
	go clean -i -r sshalama

.PHONY: test
test:
	go test -tags '$(BUILDTAGS)' -ldflags '$(LDFLAGS)' sshalama/...
