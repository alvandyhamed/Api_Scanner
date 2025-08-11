# ---------- Build stage ----------
FROM golang:1.22-bookworm AS build
WORKDIR /src
ENV GOTOOLCHAIN=go1.24.5+auto


COPY go.mod go.sum ./
RUN go mod download


COPY . .


RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /out/sitechecker ./.

# ---------- Runtime stage ----------
FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive

# Chromium + فونت‌ها + CA
RUN apt-get update && apt-get install -y --no-install-recommends \
    chromium \
    fonts-liberation \
    fonts-noto-color-emoji \
    ca-certificates \
    tzdata \
    curl \
 && rm -rf /var/lib/apt/lists/*


ENV CHROME_BIN=/usr/bin/chromium


COPY --from=build /out/sitechecker /usr/local/bin/sitechecker

EXPOSE 8080


ENTRYPOINT ["/usr/local/bin/sitechecker"]
