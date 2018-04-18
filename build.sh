#!/bin/sh
go get github.com/josephspurrier/goversioninfo/cmd/goversioninfo
go generate
go build -o dap.exe .
