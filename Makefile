MODULE  := github.com/eu-sovereign-cloud/iam
BINARY  := iamd
IMAGE   := iam:latest

.PHONY: build
build:
	CGO_ENABLED=0 go build -o bin/$(BINARY) ./cmd/iamd

.PHONY: run
run:
	go run ./cmd/iamd

.PHONY: test
test:
	go test ./... -race -count=1

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: fmt
fmt:
	gofmt -l -w .
	goimports -l -w .

.PHONY: fmt-check
fmt-check:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)

.PHONY: vet
vet:
	go vet ./...

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: vuln
vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

.PHONY: e2e
e2e:
	go test -tags e2e ./test/e2e/... -v -timeout 5m

.PHONY: docker-build
docker-build:
	docker build -t $(IMAGE) .

.PHONY: clean
clean:
	rm -rf bin
