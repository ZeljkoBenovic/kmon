run:
	go run cmd/main.go

build:
	go build -ldflags "-s -w" -o kmon cmd/main.go