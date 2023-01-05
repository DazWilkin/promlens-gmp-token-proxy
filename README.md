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

|Metric|Type|Description|
|------|----|----------|
|`proxied_total`|Counter|Number of requests that have been proxied|
|`proxied_error`|Counter|Number of requests that failed to be proxied|
|`tokens_total`|Counter|Number of tokens that have been minted|
|`tokens_error`|Counter|Number of tokens that failed to be minted|