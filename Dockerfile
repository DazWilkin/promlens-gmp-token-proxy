ARG GOLANG_VERSION="1.25.3"
ARG PROJECT="promlens-gmp-token-proxy"

ARG TARGETOS
ARG TARGETARCH

ARG COMMIT
ARG VERSION

FROM --platform=${TARGETARCH} docker.io/golang:${GOLANG_VERSION} AS build

ARG PROJECT

WORKDIR /${PROJECT}

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

COPY cmd/proxy cmd/proxy

ARG TARGETOS
ARG TARGETARCH

ARG COMMIT
ARG VERSION

RUN BUILD_TIME=$(date +%s) && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -a \
    -installsuffix cgo \
    -ldflags "-X 'main.BuildTime=${BUILD_TIME}' -X 'main.GitCommit=${COMMIT}' -X 'main.OSVersion=${VERSION}'" \
    -o /bin/proxy \
    ./cmd/proxy


FROM --platform=${TARGETARCH} gcr.io/distroless/static-debian12:latest

ARG PROJECT

LABEL org.opencontainers.image.source=https://github.com/DazWilkin/${PROJECT}

COPY --from=build /bin/proxy /

ENTRYPOINT ["/proxy"]
CMD ["--prefix","--proxy=0.0.0.0:7777","--remote=0.0.0.0:9090"]
