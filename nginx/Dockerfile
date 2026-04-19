# nginx/Dockerfile — Multi-stage build: frontend SPA + admin SPA + nginx
# Stage 1: build frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /build/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ .
RUN npm run build

# Stage 2: build admin (base: /admin/ 이미 vite.config.ts에 설정됨)
FROM node:20-alpine AS admin-builder
WORKDIR /build/admin
COPY admin/package*.json ./
RUN npm ci
COPY admin/ .
RUN npm run build

# Stage 3: nginx
FROM nginx:1.27-alpine
COPY --from=frontend-builder /build/frontend/dist /usr/share/nginx/html
COPY --from=admin-builder /build/admin/dist /usr/share/nginx/admin
# nginx.conf은 K8s ConfigMap에서 mount (/etc/nginx/conf.d/default.conf)
EXPOSE 80
