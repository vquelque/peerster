#!/bin/bash
cd /Users/Valentin/go
find src/github.com/vquelque/Peerster -name '*.go' | tar -zcvf /Users/Valentin/Downloads/src.tar.gz -T - 