BUILD_DATE := `date -u +%Y-%m-%d`
BUILD_GIT  := `git rev-parse --short HEAD`
FLAGS      := -ldflags "-X main.build=$(BUILD_GIT) -X main.date=$(BUILD_DATE)"

.PHONY: build debug run

run: debug

debug:
	@echo "debug build..."
	@go build -race $(FLAGS)
	@echo "run..."
	@./pusher -address localhost:8080

build:
	@echo "build..."
	@go build $(FLAGS)