.PHONY: test acceptance acceptance-docker lint fmt

# Unit + lifecycle tests (real git repos, race detector).
test:
	go test -race ./...

# Acceptance tests locally — builds grove from source automatically.
acceptance:
	go test -v -count=1 -timeout=300s ./acceptance/...

# Acceptance tests in Docker — isolated, CI-identical.
acceptance-docker:
	docker build -f acceptance/Dockerfile -t grove-acceptance .
	docker run --rm grove-acceptance

fmt:
	gofmt -l .

lint:
	go vet ./...
