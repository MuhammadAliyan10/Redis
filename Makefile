.PHONY: build run test clean

APP_NAME = redis-server
REPLICA_NAME = redis-replica
SENTINEL_NAME = redis-sentinel

# Build all binaries
build:
	go build -o bin/$(APP_NAME) cmd/redis-server/main.go
	go build -o bin/$(REPLICA_NAME) cmd/replica/main.go
	go build -o bin/$(SENTINEL_NAME) cmd/sentinel/main.go

# Run the master server
run: build
	./bin/$(APP_NAME) --port 6379

# Run a replica server
run-replica: build
	./bin/$(APP_NAME) --port 6380 --replicaof 127.0.0.1:6379

# Run sentinel
run-sentinel: build
	./bin/$(SENTINEL_NAME)

# Run tests
test:
	go test -v -race ./...

# Format code
fmt:
	go fmt ./...

# Clean up binaries and logs
clean:
	rm -rf bin/
	rm -f *.aof
