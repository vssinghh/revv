FROM golang:1.26.4-alpine
RUN apk add --no-cache git make bash curl docker-cli
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make build
