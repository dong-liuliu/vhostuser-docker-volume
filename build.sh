#!/bin/bash

version=0.1
time=$(date "+%x-%R")

go build -ldflags="-X 'main.Version=${version}' -X 'main.BuildTime=${time}'"