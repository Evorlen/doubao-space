.PHONY: dev-backend dev-frontend frontend-build build build-darwin-arm64 build-darwin-amd64 test clean

APP_NAME := doubao-space
GO_ENTRY := ./cmd/doubao-space
FRONTEND_DIR := web/frontend

dev-backend:
	go run $(GO_ENTRY) --dev --addr 127.0.0.1:8081

dev-frontend:
	cd $(FRONTEND_DIR) && npm run dev

frontend-build:
	cd $(FRONTEND_DIR) && npm ci && npm run build

build: frontend-build
	go build -trimpath -ldflags="-s -w" -o build/$(APP_NAME) $(GO_ENTRY)

build-darwin-arm64: frontend-build
	@mkdir -p build
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o build/$(APP_NAME)-darwin-arm64 $(GO_ENTRY)

build-darwin-amd64: frontend-build
	@mkdir -p build
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o build/$(APP_NAME)-darwin-amd64 $(GO_ENTRY)

test:
	go test ./...

clean:
	@rm -rf build
