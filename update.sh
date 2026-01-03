#! /bin/zsh

go get -v -u ./...
git add go.mod go.sum
git ci -m 'update go deps'

go run ./cmd/uploader

git add wallpapers.db
git ci -m 'update wallpapers.db'

git push -u
