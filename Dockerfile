
FROM golang:1.19

WORKDIR /src
COPY . ./

# required to exec provider plugins
ENV CGO_ENABLED=0
RUN go build

ENTRYPOINT ["/src/terraform-resource-markdown-table-action"]
