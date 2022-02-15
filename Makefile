.PHONY: all deps clean

all: cmd/hueliod/hueliod

legal_notices.go:
	go mod tidy
	go generate ./...

deps: frontend/node-modules
	go get -v ./...

cmd/hueliod/hueliod: frontend/dist
	go get ./...
	cd cmd/hueliod/ && go build

frontend/dist:
	cd frontend && yarn dist

frontend/node-modules: frontend/package.json frontend/yarn.lock
	cd frontend && yarn install --frozen-lockfile 

clean:
	rm -rf cmd/hueliod/hueliod
	rm -rf frontend/dist
