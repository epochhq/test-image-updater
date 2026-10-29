FROM docker.io/library/golang:1.27.3-alpine@sha256:d200d2dd76b03a99b52651fd1325aff7dcd35422c35c01bac56571b0507f3deb AS dependencies

ENV GOTOOLCHAIN=local GOWORK=off
WORKDIR /app

COPY go.mod go.sum ./
RUN test "$(go env GOVERSION)" = "go$(go list -m -f '{{.GoVersion}}')"
RUN go mod download

FROM dependencies AS builder
COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown

RUN test "$(go env GOVERSION)" = "go$(go list -m -f '{{.GoVersion}}')" && \
    CGO_ENABLED=0 GOOS=linux go build -mod=readonly \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}" \
    -o server .

FROM gcr.io/distroless/static-debian13:nonroot@sha256:d62067879de890a1def9da0e7766b7d20810a5e5555cef8506b752437d7a3e88

COPY --from=builder /app/server /server

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]
