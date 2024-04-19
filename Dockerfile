FROM golang:1.21 AS build

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

COPY internal/ internal/
COPY cmd/gitops-actions/main.go .

ARG VERSION
ARG SOURCE_COMMIT
ARG SOURCE_BRANCH
ARG BUILD_DATE
ARG BUILD_USER

ARG TARGETPLATFORM

RUN CGO_ENABLED=0 GOOS=linux GOARCH=$(echo ${TARGETPLATFORM} | cut -d / -f2) go build -a -o gitops-actions -ldflags " \
    -X gitops-actions/internal/version.Version=${VERSION} \
    -X gitops-actions/internal/version.Revision=${SOURCE_COMMIT} \
    -X gitops-actions/internal/version.Branch=${SOURCE_BRANCH} \
    -X gitops-actions/internal/version.BuildDate=${BUILD_DATE} \
    -X gitops-actions/internal/version.BuildUser=${BUILD_USER}"

# Use distroless as minimal base image
# Refer to https://github.com/GoogleContainerTools/distroless for more details.
FROM gcr.io/distroless/static:nonroot
COPY --from=build /workspace/gitops-actions /

USER 65532:65532

ENTRYPOINT ["/gitops-actions"]
