# --- build ---
FROM node:20-alpine AS build
WORKDIR /app

# فقط فایل‌های پکیج؛ اگر lock بود استفاده می‌کنیم
COPY package.json package-lock.json* ./

# اگر lockfile هست npm ci، وگرنه npm install
RUN if [ -f package-lock.json ]; then \
      npm ci --no-audit --no-fund; \
    else \
      npm install --no-audit --no-fund; \
    fi

# بقیهٔ سورس
COPY . .
RUN npm run build

# --- run ---
FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
