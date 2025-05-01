FROM golang:1.24.2 AS build

WORKDIR /app
COPY . ./
RUN go mod tidy
RUN go build -o app ./bot.go

FROM alpine:latest
#this seems dumb, but the libc from the build stage is not the same as the alpine libc
#create a symlink to where it expects it since they are compatable. https://stackoverflow.com/a/35613430/3105368
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
WORKDIR /app
COPY --from=build /app/app ./
RUN apk add docker-cli

EXPOSE 8080

# Command to run the discord bot. Ensure to mount a .env
CMD ["./app"]
