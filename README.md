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

## Development

```bash
cd cmd/hueliod/frontend
yarn install
yarn dev
```

```bash
cd cmd/hueliod
go run main.go -bind localhost:8080 -cors -store secrets.txt
```

## License

Licensed under MIT