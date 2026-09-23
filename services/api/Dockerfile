FROM golang:1.23-alpine AS build
WORKDIR /src
COPY pkg/ pkg/
COPY services/api/go.mod services/api/go.sum services/api/
WORKDIR /src/services/api
RUN go mod download
COPY services/api/ .
RUN for b in api worker migrate; do \
      CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/$b ./cmd/$b || exit 1; \
    done

FROM alpine:3.20
RUN apk add --no-cache ca-certificates poppler-utils && adduser -D -u 10001 app
COPY --from=build /out/ /app/
USER app
ENTRYPOINT ["/app/api"]
