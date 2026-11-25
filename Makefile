BINARY := mayamcp
STATICCHECK_VERSION := v0.8.2
VERSION ?= dev
MODULE ?= newmodule
TASKDEV_CLI ?= $(HOME)/.config/opencode/skills/taskdev/scripts/taskdev-cli.ts

.PHONY: build test check cover clean rename taskdev-sync

build:
	go build -ldflags "-X main.defaultVersion=$(VERSION)" -o $(BINARY) ./cmd/mayamcp

test:
	go test -count=1 ./...

cover:
	go test -count=1 -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

check:
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "not gofmt-clean:"; echo "$$out"; exit 1; fi
	go vet ./...
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	else \
		go run honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION) ./...; \
	fi
	go test -count=1 -race ./...

clean:
	rm -f $(BINARY) coverage.out

# Instantiate the template under a new Go module path. Review the diff afterwards.
rename:
	@test -n "$(MODULE)" || (echo "MODULE is required" && exit 1)
	@grep -rl --include='*.go' --include='*.md' 'mayamcp' . | xargs sed -i 's/mayamcp/$(MODULE)/g'
	@echo "renamed to $(MODULE); run make check and review the diff"

# Regenerate the TaskDev index, then undo the CLI's hardcoded fallback name.
taskdev-sync:
	@bun run $(TASKDEV_CLI) sync
	@grep -q 'proyek \*\*MayaMCP\*\*' TaskDev/README.md || \
		sed -i 's/proyek \*\*[^*]*\*\*/proyek **MayaMCP**/' TaskDev/README.md
