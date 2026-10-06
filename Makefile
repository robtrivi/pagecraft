.PHONY: test cover vet bench

test:
	go test ./...
	go vet ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -5

bench:
	go test ./render/ -run XXX -bench 'BenchmarkRender'
