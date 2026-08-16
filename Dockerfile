# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM golang:1.26.6 AS deps

WORKDIR /src

# Download Go dependencies in a dedicated layer so source changes do not
# invalidate the module cache.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	go mod download

FROM deps AS build

WORKDIR /src

# Cross-compilation targets come from buildx; they default to the build
# platform for a plain `docker build`. Hardcoding the architecture here would
# put amd64 binaries into arm64-labeled manifests as soon as the release
# workflow builds more than one platform.
ARG TARGETOS
ARG TARGETARCH

# Copy only the source tree required to build the application binary.
COPY cmd ./cmd
COPY internal ./internal

# Build a static binary and strip debug symbols to keep the runtime image small.
RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
	go build -trimpath -ldflags="-s -w" -o /out/tg_stt_bot ./cmd/app

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Runtime image contains only the compiled binary. The bot configuration must be
# mounted from outside, e.g. -config /etc/tg-stt-bot/config.json.
COPY --from=build /out/tg_stt_bot /app/tg_stt_bot

USER nonroot:nonroot

ENTRYPOINT ["/app/tg_stt_bot"]
