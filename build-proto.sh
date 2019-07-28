#!/bin/sh

protoc --go_out="$GOPATH/src" proto/*.proto