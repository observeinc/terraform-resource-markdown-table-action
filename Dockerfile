
FROM golang:1.19-alpine

RUN apk add --no-cache git

WORKDIR /src
COPY . ./

RUN CGO_ENABLED=0 go build

ENTRYPOINT ["/src/terraform-resource-markdown-table-action"]
