all: test build

test:
	go test -cover ./...

build:
	go build -o trainer ./cmd/neatflappy-trainer
	go build -o human ./cmd/neatflappy-human
	go build ./cmd/neatflappy