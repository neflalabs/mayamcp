BINARY := mayamcp
STATICCHECK_VERSION := v0.8.2

.PHONY: build test check clean

build:
	go build -o $(BINARY) ./cmd/mayamcp

test:
	go test -count=1 ./...

check:
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "not gofmt-clean:"; echo "$$out"; exit 1; fi
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION) ./...
	go test -count=1 -race ./...

clean:
	rm -f $(BINARY)
