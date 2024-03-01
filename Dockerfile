ARG GOLANG_VERSION="1.22"
ARG PROJECT="promlens-gmp-token-proxy"

ARG COMMIT
ARG VERSION

FROM docker.io/golang:${GOLANG_VERSION} as build

ARG PROJECT

ARG COMMIT
ARG VERSION

WORKDIR /${PROJECT}

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

COPY cmd/proxy cmd/proxy

RUN BUILD_TIME=$(date +%s) && \
    CGO_ENABLED=0 GOOS=linux go build \
    -a \
    -installsuffix cgo \
    -ldflags "-X 'main.BuildTime=${BUILD_TIME}' -X 'main.GitCommit=${COMMIT}' -X 'main.OSVersion=${VERSION}'" \
    -o /bin/proxy \
    ./cmd/proxy


FROM scratch

ARG PROJECT

LABEL org.opencontainers.image.source https://github.com/DazWilkin/${PROJECT}

COPY --from=build /bin/proxy /
COPY --from=build /etc/passwd /etc/passwd
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENTRYPOINT ["/proxy"]
CMD ["--prefix","--proxy=0.0.0.0:7777","--remote=0.0.0.0:9090"]