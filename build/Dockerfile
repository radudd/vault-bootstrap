FROM golang:alpine as build

# set labels for metadata
LABEL maintainer="Radu Domnu<rdomnu@redhat.com>" \
  name="vault-bootstrap" \
  description="A Kubernetes terminating container for bootstraping Vault" 

# set environment variables
ENV GO111MODULE=on \
  CGO_ENABLED=0

RUN apk add git make 

# Test and build 
WORKDIR /build
COPY . .
RUN make test
RUN make build

FROM scratch

WORKDIR /app

COPY --from=build /build/vault-bootstrap .

CMD ["/app/vault-bootstrap"]

USER 1001