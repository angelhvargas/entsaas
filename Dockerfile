# Stage 1: Build frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app/web
COPY web/package.json web/package-lock.json* ./
RUN npm ci
COPY web/ .
RUN npm run build

# Stage 2: Build backend
FROM golang:1.26.3-alpine AS backend-builder
WORKDIR /app
RUN apk add --no-cache git make
ARG VERSION=dev
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -buildvcs=false -trimpath -ldflags="-s -w" -o api ./cmd/api && \
    go build -buildvcs=false -trimpath -ldflags="-s -w" -o migrate ./cmd/migrate && \
    go build -buildvcs=false -trimpath -ldflags="-s -w" -o seed ./cmd/seed

# Stage 3: Final image
FROM alpine:3.21 AS final
ARG VERSION=dev
LABEL org.opencontainers.image.title="entsaas" \
      org.opencontainers.image.description="EntSaaS application image" \
      org.opencontainers.image.version="${VERSION}"

WORKDIR /app
RUN addgroup -S entsaas && adduser -S -u 10001 -G entsaas entsaas
RUN apk upgrade --no-cache

COPY --from=backend-builder --chown=entsaas:entsaas /app/api .
COPY --from=backend-builder --chown=entsaas:entsaas /app/migrate .
COPY --from=backend-builder --chown=entsaas:entsaas /app/seed .
COPY --from=frontend-builder --chown=entsaas:entsaas /app/web/dist ./web/dist

USER entsaas
EXPOSE 8080
CMD ["./api"]
