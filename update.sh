#! /bin/zsh

go get -v -u -d ./...
git add go.mod go.sum
git ci -m 'update go deps'
git push

go run ./cmd/uploader
