test:
	go test -trimpath ./...

ci:
	go test -race -trimpath ./...

lint:
	golangci-lint run --config .golangci.yml

release:
	go run github.com/kevinburke/bump_version@latest --tag-prefix=v minor version/version.go
