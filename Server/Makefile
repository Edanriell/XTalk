.PHONY: help proto proto-user build run-dev run-prod docker-up docker-down clean test

help:
	@echo "Available commands:"
	@echo "  make proto       - Generate protobuf code"
	@echo "  make proto-user  - Generate user-service protobuf code"
	@echo "  make build       - Build all services"
	@echo "  make run-dev     - Run all services locally"
	@echo "  make run-prod    - Run with Docker Compose"
	@echo "  make docker-up   - Start Docker containers"
	@echo "  make docker-down - Stop Docker containers"
	@echo "  make clean       - Clean build artifacts"
	@echo "  make test        - Run tests"

proto:
	@echo "Generating protobuf code..."
	@for service in auth chat matching message user; do \
		protoc -I proto --go_out=proto/$$service --go_opt=paths=source_relative \
			--go-grpc_out=proto/$$service --go-grpc_opt=paths=source_relative \
			proto/$$service.proto; \
	done

proto-user:
	protoc -I proto --go_out=proto/user --go_opt=paths=source_relative \
		--go-grpc_out=proto/user --go-grpc_opt=paths=source_relative \
		proto/user.proto

build:
	@echo "Building all services..."
	go build -o bin/auth-service ./auth-service
	go build -o bin/user-service ./services/user
	go build -o bin/chat-service ./chat-service
	go build -o bin/message-service ./message-service
	go build -o bin/api-gateway ./api-gateway

run-dev:
	@echo "Starting services locally..."
	@echo "Make sure PostgreSQL and Redis are running!"
	@make build
	@echo "Starting auth-service..."
	./bin/auth-service &
	@echo "Starting user-service..."
	./bin/user-service &
	@echo "Starting chat-service..."
	./bin/chat-service &
	@echo "Starting message-service..."
	./bin/message-service &
	@echo "Starting api-gateway..."
	./bin/api-gateway

run-prod: docker-up

docker-up:
	@echo "Starting Docker containers..."
	docker-compose up --build -d
	@echo "Services are running!"
	@echo "API Gateway: http://localhost:8080"
	@echo "Health check: http://localhost:8080/health"

docker-down:
	@echo "Stopping Docker containers..."
	docker-compose down

docker-logs:
	docker-compose logs -f

clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	docker-compose down -v

test:
	@echo "Running tests..."
	go test -v ./...

install-deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

migrate-up:
	@echo "Running migrations..."
	psql -U postgres -d connect_db -f migrations/001_init.sql

generate-proto:
	@echo "Installing protoc plugins..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@make proto
