# Build a partir da raiz do repositório:
#   docker build -f apps/web/Dockerfile .
FROM node:20-alpine AS build
WORKDIR /app
COPY apps/web/package.json apps/web/package-lock.json ./
RUN npm ci
COPY apps/web/ .
RUN npm run build

FROM nginx:1.27-alpine
ENV PORT=8080 API_URL=http://host.docker.internal:8080 API_HOST=localhost
COPY apps/web/nginx/default.conf.template /etc/nginx/templates/default.conf.template
COPY --from=build /app/dist /usr/share/nginx/html
