# A token proxy to authenticate [PromLens](https://promlens.com/) to [Google Managed Prometheus](https://cloud.google.com/stackdriver/docs/managed-prometheus)

[![build-container](https://github.com/DazWilkin/promlens-gmp-token-proxy/actions/workflows/build.yml/badge.svg)](https://github.com/DazWilkin/promlens-gmp-token-proxy/actions/workflows/build.yml)

![Google Compute Engine: Instance CPU Utilization](/images/promlens.compute_googleapis._com.png)

## GCP

```bash
PROJECT="..."
ACCOUNT="prom-proxy"

EMAIL=${ACCOUNT}@${PROJECT}.iam.gserviceaccount.com

gcloud iam service-accounts create ${ACCOUNT} \
--project=${PROJECT}

gcloud iam service-accounts keys create ${PWD}/${ACCOUNT}.json \
--iam-account=${EMAIL} \
--project=${PROJECT}

gcloud projects add-iam-policy-binding ${PROJECT} \
--member=serviceAccount:${EMAIL} \
--role=roles/monitoring.viewer

export GOOGLE_APPLICATION_CREDENTIALS=${PWD}/${ACCOUNT}.json
```

## Run

```bash
PROJECT="..."

PORT="..."

go run ./cmd/proxy \
--remote="monitoring.googleapis.com" \
--prefix="/v1/projects/${PROJECT}/location/global/prometheus" \
--proxy="0.0.0.0:${PORT}"
```

## Test

```bash
PROJECT="..."

REMOTE="https://monitoring.googleapis.com/v1/projects/${PROJECT}/location/global/prometheus"
TOKEN="$(gcloud auth print-access-token)"

PORT="..."
PROXY="http://0.0.0.0:${PORT}"

# Direct
curl \
--get \
--header "Authorization: Bearer ${TOKEN}" \
--data-urlencode "query=1" \
${REMOTE}/api/v1/query

# Proxied
curl \
--get \
--data-urlencode "query=1" \
${PROXY}/api/v1/query
```

## Metrics

All metrics are prefixed: `promlens_gmp_token_proxy_`

|Metric|Type|Labels|Description|
|------|----|------|-----------|
|`proxied_status`|Counter|`code`|Number of proxied requests that returned a status code|
|`proxied_total`|Counter||Number of proxied requests|
|`proxied_error`|Counter||Number of proxied requests that errored|
|`tokens_total`|Counter||Number of token requests|
|`tokens_error`|Counter||Number of token requests that failed|

Proxied requests may fail and return an error. Proxied requqests may succeed but include a HTTP status code that represents client (`4XX`) or server (`5XX`) errors. The `proxied_` metrics attempt to reflect this nuance.

## [Sigstore](https://www.sigstore.dev/)

`promlens-gmp-token-proxy` container images are being signed by Sigstore and may be verified:

```bash
cosign verify \
--key=./cosign.pub \
ghcr.io/dazwilkin/promlens-gmp-token-proxy:ecb6313799de1ebd3c354590259b751082fc387d
```

NOTE cosign.pub may be downloaded [here](./cosign.pub)

To install cosign, e.g.:

```bash
go install github.com/sigstore/cosign/cmd/cosign@latest
```