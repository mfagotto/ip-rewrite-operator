# Build the operator manager (webhook + controller) on UBI 10.
# Digests pinned for reproducible, VA-reviewable builds.
#
# Builder: ubi10/go-toolset 1.27.4 (1795507689)
# Runtime: ubi10/ubi-micro (static binary only)

FROM registry.access.redhat.com/ubi10/go-toolset@sha256:b9f867772f02ef0c8167b1ac327b615c9c6b06be16f5a42ea322597633d1d25e AS builder
ARG TARGETOS=linux
ARG TARGETARCH=amd64
USER 0
WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download
COPY api/ api/
COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -a -o manager ./cmd/

FROM registry.access.redhat.com/ubi10/ubi-micro@sha256:2ad0f98cb13160c4773c945ef93f88b25831c7f27cd3780d72a7e78b210ea72b
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532
ENTRYPOINT ["/manager"]
