# huelio

A very simple app to control your hue lights that starts much faster than the mobile apps.

Built using Go, JavaScript, Font-Awesome & friends. 

## Production

Requirements:
- [go](https://golang.org/)
- [yarn](https://yarnpkg.com/)
- [make](https://www.gnu.org/software/make/)

```bash
make deps # once, to install dependencies
make all # to build a prod executable
```

### Legal Notices

To update legal notices, run the following:

```bash
go install github.com/tkw1536/gogenlicense/cmd/gogenlicense@latest
make -B legal.go
```

Then commit the result.

## Deployment

![CI Status](https://github.com/tkw1536/huelio/workflows/docker/badge.svg)

Available as a Docker Image on [GitHub Packages](https://github.com/tkw1536/huelio/pkgs/container/hueliod).
Automatically built on every commit.

```bash
 docker run -ti -v credentials:/data/ -p 8080:8080 ghcr.io/tkw1536/hueliod
```

## Development

```bash
cd frontend
yarn install
yarn dev
```

```bash
cd cmd/hueliod
go run main.go -debug -store secrets.txt
```

## License

Licensed under MIT