BIN_NAME=fileStorage

all: test build 

run:
	go run *.go

build:
	go build -o ${BIN_NAME} main.go

test:
	go test -v -race ./...

clean:
	go clean
