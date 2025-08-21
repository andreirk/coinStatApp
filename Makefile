.PHONY: build run test clean docker-build docker-run start-deps stop-deps

# Build the app
build:
	go build -o bin/coinStatApp ./cmd/coinStatApp

# Run the app
run: build
	./bin/coinStatApp

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Build Docker image
docker-build:
	docker build -t coinStatApp .

# Run in Docker
docker-run:
	docker run --rm -p 8080:8080 coinStatApp

# Start dependencies (Redis, ClickHouse)
start-deps:
	docker-compose up -d redis clickhouse

# Stop dependencies
stop-deps:
	docker-compose down

# Full start with dependencies
up: start-deps run

# Full docker environment
docker-up:
	docker-compose up -d
