test:
	go test -trimpath ./...

ci:
	go test -race -trimpath ./...

lint:
	golangci-lint run --config .golangci.yml
