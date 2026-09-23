FROM golang:1.25-alpine AS build
WORKDIR /src
COPY pkg/ pkg/
COPY services/scraper/go.mod services/scraper/
WORKDIR /src/services/scraper
RUN go mod download
COPY services/scraper/ .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/scraper ./cmd/scraper

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/scraper /app/scraper
ENTRYPOINT ["/app/scraper"]
