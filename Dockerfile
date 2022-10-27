
FROM golang:1.19-alpine

RUN apk add --no-cache git

WORKDIR /src
COPY . ./

# required to exec provider plugins
ENV CGO_ENABLED=0
RUN go build

ENTRYPOINT ["/src/terraform-resource-markdown-table-action"]
