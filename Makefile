run:
	go run cmd/main.go -h

build:
	go build -ldflags "-s -w" -o kmon cmd/main.go