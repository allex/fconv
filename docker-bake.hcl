# by allex_wang

function image_name {
  params = [prefix, name]
  result = notequal("", prefix) ? "${prefix}/${name}" : "${name}"
}

variable "NAME" {
  default = "fconv"
}

variable "PREFIX" {
  default = "harbor.tidu.io/tdio"
}

variable "BUILD_TAG" {
  default = "0.0.0"
}

variable "DOCKER_TAG" {
  default = "latest"
}

group "default" {
  targets = ["main"]
}

target "main" {
  context = "."
  dockerfile = "Dockerfile"
  args = {
    BUILD_TAG = "${BUILD_TAG}"
  }
  tags = [
    "${image_name(PREFIX, NAME)}:${DOCKER_TAG}",
    "${image_name(PREFIX, NAME)}:${BUILD_TAG}"
  ]
  platforms = ["linux/amd64","linux/arm64","linux/386"]
}
