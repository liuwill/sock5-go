install:
	go get

server:
	cd bin/ && go run main.go

.PHONY: server
