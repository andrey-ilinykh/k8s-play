#!/bin/bash

set -e

CGO_ENABLED=0 go build -o main .

docker build -t localhost:5000/backend .

docker push localhost:5000/backend