package main

//go:generate go run github.com/swaggo/swag/cmd/swag@v1.16.2 init --parseDependency --parseInternal --parseDepth 2 --dir .,../../internal/rides --generalInfo main.go --output ../../docs/rides --instanceName rides --outputTypes go,yaml,json
