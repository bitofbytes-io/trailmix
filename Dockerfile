# syntax=docker/dockerfile:1@sha256:572cebbc2b15e6f84c077742952674889cb63cd5e9b80abdffcbe587b6f19183

FROM golang:1.27.4-alpine3.23@sha256:f70850ef6ce341e8b6427f677049145d63c0b7ce002d4d7e3d464828b1b96dcd AS builder
ARG VERSION=dev
ARG REVISION=unknown
ARG TARGETARCH
WORKDIR /src

# libstdc++ is required by the Tailwind musl binary.
RUN apk add --no-cache ca-certificates curl libstdc++ libgcc

# Tailwind's musl build, because this stage is Alpine. Checksums are the
# v4.3.0 release assets. TARGETARCH keeps arm64 image builds working.
RUN set -eu; \
    case "${TARGETARCH}" in \
      amd64) tw_arch=x64-musl; tw_sha=d861210e7c772e3b8a8b302d51bd4c8e2ab2e6f5dcb7545b16da6e9676c99079 ;; \
      arm64) tw_arch=arm64-musl; tw_sha=9c106e815ce7ea99a65f91c13be2c51f00388dd3c0127c7a13a38f76cd1e26aa ;; \
      *) echo "unsupported architecture ${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    curl -fsSL "https://github.com/tailwindlabs/tailwindcss/releases/download/v4.3.0/tailwindcss-linux-${tw_arch}" -o /usr/local/bin/tailwindcss; \
    echo "${tw_sha}  /usr/local/bin/tailwindcss" | sha256sum -c -; \
    chmod +x /usr/local/bin/tailwindcss

COPY go.mod go.sum ./
RUN go mod download

# Generated templ and CSS are untracked. Build them here so `docker compose up`
# works from a fresh clone. templ version matches go.mod.
COPY . .
RUN go run github.com/a-h/templ/cmd/templ@v0.3.1020 generate \
 && /usr/local/bin/tailwindcss -i ./tailwind/styles.css -o ./static/styles.css --minify

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.version=${VERSION} -X main.revision=${REVISION}" -o /out/trailmix ./cmd/trailmix \
 && CGO_ENABLED=0 GOOS=linux go build -o /out/trailmix-migrate ./cmd/trailmix-migrate

FROM alpine:3.24.4@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf
ARG VERSION=dev
ARG REVISION=unknown
ARG SOURCE_URL=https://github.com/bitofbytes-io/trailmix
WORKDIR /app

# wget is what the Compose healthcheck runs. Busybox in this image does not
# provide it.
RUN apk add --no-cache ca-certificates tzdata wget

COPY --from=builder /out/trailmix ./trailmix
# A one-off migrate service runs this before the app on `docker compose up`.
COPY --from=builder /out/trailmix-migrate ./trailmix-migrate
COPY --from=builder /src/static ./static
COPY migrations ./migrations

RUN addgroup -S trailmix \
    && adduser -S -G trailmix trailmix \
    && chown -R trailmix:trailmix /app

LABEL org.opencontainers.image.source="${SOURCE_URL}" \
      org.opencontainers.image.revision="${REVISION}" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.title="trailmix" \
      org.opencontainers.image.description="Trail Mix web application"

ENV PORT=4600
ENV LOG_LEVEL=info
USER trailmix

EXPOSE 4600
CMD ["./trailmix"]
