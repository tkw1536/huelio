FROM docker.io/library/alpine:3 as os

# install ca-certificates
RUN apk add --update --no-cache ca-certificates

# create www-data
RUN set -x ; \
  addgroup -g 82 -S www-data ; \
  adduser -u 82 -D -S -G www-data www-data && exit 0 ; exit 1

# build the frontend
FROM docker.io/library/node:16 as frontend
ADD frontend /app/frontend/
WORKDIR /app/frontend/
RUN yarn install --frozen-lockfile
RUN yarn dist

# build the backend
FROM docker.io/library/golang:1.17 as builder
ADD . /app/
WORKDIR /app/
COPY --from=frontend /app/frontend/dist /app/frontend/dist
RUN go get ./cmd/hueliod
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o hueliod ./cmd/hueliod

# add it into a scratch image
FROM alpine:3

# add the user
COPY --from=os /etc/passwd /etc/passwd
COPY --from=os /etc/group /etc/group

# grab ssl certs
COPY --from=os /etc/ssl/certs /etc/ssl/certs

# create a volume at /data/
RUN mkdir /data/ && chown -R www-data:www-data /data/
VOLUME /data/

# add the app
COPY --from=builder /app/hueliod /hueliod


# and set the entry command
EXPOSE 8080
USER www-data:www-data
CMD ["/hueliod", "-bind", "0.0.0.0:8080", "-store", "/data/secrets.txt"]