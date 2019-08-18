
release: 
	./release.sh

test:
	go test ./...

.PHONY: install test build docker release
