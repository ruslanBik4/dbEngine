BRANCH=`git branch --show-current`
VERSION=`git describe --tags`

define increment
	$(eval v := $(shell git describe --tags --abbrev=0 | sed -Ee 's/^v|-.*//'))
    $(eval n := $(shell echo $(v) | awk -F. -v OFS=. -v f=$1 '{ $$f++ } 1'))
    @git tag -a v$(n) -m "Bumped to version $(n), $(m)"
	@git push
	@git push --tags
	@echo "Updating version $(v) to $(n)"
endef

.PHONY : all
all : git patch

release: dep proto-generate git patch

tests:
	go test -v ./...

test_cover:
	go test ./... -cover

git:
	git add .
	git commit -m "$m"
	git push -u origin ${BRANCH}

patch:
	$(call increment,3,path)

lint: check-lint dep
	golangci-lint run --timeout=5m -c .golangci.yml

check-lint:
	@which golangci-lint || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.25.0

dep:
	@go mod tidy
	@go mod download