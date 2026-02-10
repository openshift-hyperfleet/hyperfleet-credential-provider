ARG BASE_IMAGE=registry.access.redhat.com/ubi9/ubi-minimal:latest

FROM golang:1.24 AS builder

ARG GIT_SHA=unknown
ARG GIT_DIRTY=""

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux make build

FROM ${BASE_IMAGE}

WORKDIR /app

COPY --from=builder /build/bin/hyperfleet-credential-provider /app/hyperfleet-credential-provider

COPY --from=builder /build/examples/kubeconfig /app/examples/kubeconfig

# Create non-root user for running the application (OpenShift best practice)
RUN microdnf install -y shadow-utils && \
    useradd -r -u 1001 -g root hyperfleet && \
    chown -R 1001:0 /app && \
    chmod -R g=u /app && \
    microdnf clean all

USER 1001

ENTRYPOINT ["/app/hyperfleet-credential-provider"]
CMD ["--help"]

LABEL name="hyperfleet-credential-provider" \
      vendor="Red Hat" \
      version="0.0.1" \
      summary="HyperFleet Credential Provider - Multi-cloud Kubernetes Token Provider" \
      description="Kubernetes authentication token provider for GKE, EKS, and AKS without cloud CLIs"
