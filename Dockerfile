<<<<<<< HEAD
FROM ubuntu:latest

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install -y \
    wget \
    git \
    openssh-client \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

RUN wget https://go.dev/dl/go1.22.4.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.22.4.linux-amd64.tar.gz && \
    rm go1.22.4.linux-amd64.tar.gz

ENV PATH=$PATH:/usr/local/go/bin

RUN mkdir -p /home/jenkins/agent

WORKDIR /home/jenkins/agent

COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

# Default command to run the agent
CMD ["agent"]
=======
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
>>>>>>> 583b24ecac91231e68664a2b7349ce19e813eab7
