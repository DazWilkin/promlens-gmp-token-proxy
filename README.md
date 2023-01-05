# 

## GCP

```bash
PROJECT="..."
ACCOUNT="prometheus-proxy"

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

|Metric|Type|Desription|
|------|----|----------|
|`proxied_total`|Counter||
|`proxied_error`|Counter||
|`tokens_total`|Counter||
|`tokens_error`|Counter||