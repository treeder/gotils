
release: 
	wget -O - https://raw.githubusercontent.com/treeder/bump/master/gitbump.sh | bash

test:
	go test ./...

.PHONY: install test build docker release
