FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.25-alpine AS api
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /commerce-ops ./cmd/api

FROM alpine:3.21
RUN addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=api /commerce-ops /app/commerce-ops
COPY --from=web /src/web/dist /app/web/dist
USER app
EXPOSE 8080
ENV STATIC_DIR=/app/web/dist
ENTRYPOINT ["/app/commerce-ops"]
