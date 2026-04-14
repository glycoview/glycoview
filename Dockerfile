# syntax=docker/dockerfile:1.7

FROM node:22-bookworm-slim AS web-build
WORKDIR /workspace/frontend

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ ./
RUN npx tsc -b && npx vite build --outDir dist

FROM golang:1.25-bookworm AS go-build
WORKDIR /workspace

ARG TARGETARCH

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=web-build /workspace/frontend/dist ./frontend/dist

RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH:-amd64} go build -trimpath -ldflags="-s -w" -o /out/glycoview ./cmd/glycoview

FROM alpine:3.22
WORKDIR /app

COPY --from=go-build /out/glycoview /app/glycoview
COPY --from=go-build /workspace/frontend/dist /app/frontend/dist

EXPOSE 8080

ENV ADDR=:8080
ENTRYPOINT ["/app/glycoview"]
