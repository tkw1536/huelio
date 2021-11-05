.PHONY: all deps clean

all: cmd/hueliod/hueliod

legal_notices.go:
	go mod tidy
	go generate ./...

deps: cmd/hueliod/frontend/node-modules
	go get -v ./...

cmd/hueliod/hueliod: cmd/hueliod/frontend/dist
	go get ./...
	cd cmd/hueliod/ && go build

cmd/hueliod/frontend/dist:
	cd cmd/hueliod/frontend && yarn dist

cmd/hueliod/frontend/node-modules: cmd/hueliod/frontend/package.json cmd/hueliod/frontend/yarn.lock
	cd cmd/hueliod/frontend && yarn install --frozen-lockfile 

clean:
	rm -rf cmd/hueliod/hueliod
	rm -rf cmd/hueliod/frontend/dist
