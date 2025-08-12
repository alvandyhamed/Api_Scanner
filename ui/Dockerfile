# syntax=docker/dockerfile:1.7

# --- build ---
FROM node:20-alpine AS build
WORKDIR /app

# رجیستری قابل تنظیم (آینه سریع‌تر هم می‌تونی بدی)
ARG NPM_REGISTRY=https://registry.npmjs.org/
RUN npm config set registry $NPM_REGISTRY \
 && npm set progress=false \
 && npm config set fund false \
 && npm config set audit false

# install
COPY package.json package-lock.json* ./
RUN --mount=type=cache,target=/root/.npm \
    if [ -f package-lock.json ]; then \
      npm ci --no-audit --no-fund; \
    else \
      npm install --no-audit --no-fund; \
    fi

# build
COPY . .
RUN npm run build

# --- run ---
FROM nginx:1.27.2-alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
