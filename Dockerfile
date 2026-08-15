# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/markdir ./cmd/markdir

FROM scratch
COPY --from=build /out/markdir /markdir
ENV MD_DIR=/docs
VOLUME ["/docs"]
EXPOSE 8080
ENTRYPOINT ["/markdir"]
