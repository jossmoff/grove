# syntax=docker/dockerfile:1
#
# Multi-stage build for grove acceptance tests.
#
# Stage 1 (build): compiles the grove binary from source.
# Stage 2 (test):  runs the godog suite against that binary in a clean
#                  environment that has git but no developer config.
#
# Usage:
#   docker build -f acceptance/Dockerfile -t grove-acceptance .
#   docker run --rm grove-acceptance

# ── Stage 1: build grove ──────────────────────────────────────────────────────
FROM golang:1.25-alpine AS build

WORKDIR /src
# Download dependencies first so this layer is cached across source changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /grove ./cmd/grove

# ── Stage 2: test runner ──────────────────────────────────────────────────────
FROM golang:1.25-alpine AS test

# git is required: grove shells out to it, and the test helpers seed repos.
RUN apk add --no-cache git

# Blank global git config so tests never fall back to the host developer's
# identity. The per-scenario .gitconfig written by World.setup() takes over.
RUN git config --global user.email "grove-test@grove.internal" && \
    git config --global user.name "Grove Test" && \
    git config --global init.defaultBranch main

# Copy the pre-built binary; GROVE_BINARY tells the test suite to use it
# directly rather than compiling from source.
COPY --from=build /grove /usr/local/bin/grove
ENV GROVE_BINARY=/usr/local/bin/grove

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

CMD ["go", "test", "-v", "-count=1", "-timeout=300s", "./acceptance/..."]
