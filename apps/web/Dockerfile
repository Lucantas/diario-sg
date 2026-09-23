FROM node:22-alpine AS build
WORKDIR /app
COPY apps/web/package.json apps/web/package-lock.json ./
RUN npm ci
COPY apps/web/ .
RUN npm run build

FROM nginx:1.27-alpine
ENV PORT=8080 API_URL=http://host.docker.internal:8080 API_HOST=localhost \
    DUMPS_URL=http://host.docker.internal:4443/diario-dumps DUMPS_HOST=localhost:4443
COPY apps/web/nginx/default.conf.template /etc/nginx/templates/default.conf.template
COPY --from=build /app/dist /usr/share/nginx/html
