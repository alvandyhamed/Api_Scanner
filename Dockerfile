# -------- Build stage --------
FROM golang:1.24.5-bookworm AS build
WORKDIR /src

# ثابت‌ها و بهینه‌سازی دانلود ماژول‌ها
ENV GOTOOLCHAIN=local \
    GOPROXY=https://proxy.golang.org,direct \
    GOSUMDB=sum.golang.org \
    CGO_ENABLED=0

# کش ماژول‌ها (نیاز به BuildKit)
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# کد
COPY . .

# بیلد چند-معماری (Buildx این ARGها رو پاس می‌دهد)
ARG TARGETOS=linux
ARG TARGETARCH=amd64
# اگر main.go در ریشه نیست مسیر را عوض کن (مثلاً ./cmd/server)
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o /out/sitechecker ./.

# -------- Runtime stage --------
FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    chromium \
    ca-certificates \
    tzdata \
    fonts-liberation \
    fonts-noto-color-emoji \
  && rm -rf /var/lib/apt/lists/*

# متغیرهای مفید
ENV CHROME_BIN=/usr/bin/chromium \
    PORT=8080 \
    LANG=C.UTF-8

# باینری
COPY --from=build /out/sitechecker /usr/local/bin/sitechecker

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/sitechecker"]
