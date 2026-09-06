# syntax=docker/dockerfile:1

FROM golang:1.26-bookworm AS api-build
WORKDIR /src/devdash
COPY --from=acc / /src/ai-command-center
COPY go.mod go.sum ./
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/devdash ./cmd/devdash

FROM debian:bookworm-slim AS api
RUN apt-get update \
	&& apt-get install -y --no-install-recommends git ca-certificates \
	&& rm -rf /var/lib/apt/lists/*
COPY --from=api-build /out/devdash /usr/local/bin/devdash
ENV HOME=/root
EXPOSE 8789
ENTRYPOINT ["devdash", "serve"]

FROM node:22-bookworm AS web-build
WORKDIR /web
RUN corepack enable
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ .
RUN pnpm build

FROM node:22-bookworm-slim AS web
WORKDIR /web
RUN corepack enable
COPY --from=web-build /web /web
ENV DEVDASH_API_URL=http://api:8789
EXPOSE 3000
CMD ["pnpm", "preview", "--host", "0.0.0.0", "--port", "3000"]
