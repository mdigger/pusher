BUILD_DATE := `date -u +%Y-%m-%d`
BUILD_GIT  := `git rev-parse --short HEAD`
FLAGS      := -ldflags "-X main.build=$(BUILD_GIT) -X main.date=$(BUILD_DATE)"

.PHONY: build debug run

run: debug

debug:
	@echo "debug build..."
	@go build -race $(FLAGS)
	@echo "remove config..."
	@rm -f test_config.gob test_store.db
	@echo "run..."
	@./pusher -config "test_config.gob" -store "memory://test_store.db" -reset -indent -monitor

build:
	@echo "build..."
	@go build $(FLAGS)