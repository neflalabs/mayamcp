BINARY := mayamcp
STATICCHECK_VERSION := v0.8.2
VERSION ?= dev
MODULE ?= newmodule
NAME := $(notdir $(MODULE))
CURRENT_MODULE := $(shell sed -n 's/^module //p' go.mod)
TASKDEV_CLI ?= $(HOME)/.config/opencode/skills/taskdev/scripts/taskdev_cli.py
RENAME_DOCS := $(shell find . -type f \( -name '*.go' -o -name '*.md' \) -not -path './.git/*' -not -path './TaskDev/*')

.PHONY: build release test check cover consumer-test clean rename taskdev-validate taskdev-sync

build:
	go build -ldflags "-X main.defaultVersion=$(VERSION)" -o $(BINARY) ./cmd/mayamcp

release:
	@mkdir -p dist
	@for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do \
		goos=$${target%/*}; goarch=$${target#*/}; \
		ext=""; [ "$$goos" = "windows" ] && ext=".exe"; \
		CGO_ENABLED=0 GOOS=$$goos GOARCH=$$goarch \
			go build -ldflags "-X main.defaultVersion=$(VERSION)" \
			-o dist/$(BINARY)-$$goos-$$goarch$$ext ./cmd/mayamcp; \
	done
	@ls -lh dist

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

# Build and test a separate Go module that imports only mayamcp/api.
consumer-test:
	cd consumer && GOWORK=off go test -count=1 ./...

clean:
	rm -f $(BINARY) coverage.out

# Instantiate the template under a new Go module path. The binary name and
# cmd/ directory follow the module basename. Review the diff afterwards.
rename:
	@test -n "$(MODULE)" || (echo "MODULE is required" && exit 1)
	@test "$(MODULE)" != "$(CURRENT_MODULE)" || (echo "module path unchanged: $(MODULE)" && exit 1)
	go mod edit -module "$(MODULE)"
	@for f in $(RENAME_DOCS); do \
		sed -i 's|mayamcp/api|$(MODULE)/api|g; s|mayamcp/internal|$(MODULE)/internal|g; s|"mayamcp"|"$(NAME)"|g; s|\./cmd/mayamcp|./cmd/$(NAME)|g; s|\./mayamcp|./$(NAME)|g' "$$f"; \
	done
	@sed -i 's|^BINARY := [A-Za-z0-9_-]*|BINARY := $(NAME)|; s|-o $$(BINARY) ./cmd/[A-Za-z0-9_-]*|-o $$(BINARY) ./cmd/$(NAME)|' Makefile
	@sed -i 's|./cmd/mayamcp|./cmd/$(NAME)|g' .github/workflows/ci.yml
	@sed -i 's|^/mayamcp|/$(NAME)|' .gitignore
	@if [ -d cmd/mayamcp ] && [ "$(NAME)" != "mayamcp" ]; then mv cmd/mayamcp cmd/$(NAME); fi
	@echo "renamed to $(MODULE) (binary $(NAME)); run make check and review the diff"

# Validate the private TaskDev notebook locally. It is intentionally not run in
# CI: TaskDev/ is gitignored, so a runner has no notebook to validate.
taskdev-validate:
	@python3 $(TASKDEV_CLI) validate --strict

# Regenerate the TaskDev index, then undo the CLI's hardcoded fallback name.
taskdev-sync:
	@python3 $(TASKDEV_CLI) sync
	@grep -q 'proyek \*\*MayaMCP\*\*' TaskDev/README.md || \
		sed -i 's/proyek \*\*[^*]*\*\*/proyek **MayaMCP**/' TaskDev/README.md
