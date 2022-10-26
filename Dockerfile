
FROM golang:1.19-alpine

RUN apk add --no-cache git

WORKDIR /src
COPY . ./

RUN go build

ENTRYPOINT ["/src/terraform-resource-markdown-table-action"]
