variable "TAG" {
  default = "latest"
}

variable "REGISTRY" {
  default = "hub.docker.com"
}

variable "BOT" {
  default = "discord-bot"
}

target "bot" {
    dockerfile = "Dockerfile"
    context = "."
    output = ["type=registry,output=registry.${REGISTRY}/${BOT}:${TAG}"]
    tags = ["${REGISTRY}/${BOT}:${TAG}"]
}

group "default" {
    targets = ["bot"]
}