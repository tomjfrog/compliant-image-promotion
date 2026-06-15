# syntax=docker/dockerfile:1

# ---------- Stage 1: build the front-end (Vite) ----------
FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package.json web/package-lock.json* ./
RUN npm install
COPY web/ ./
RUN npm run build

# ---------- Stage 2: build the Go back-end ----------
FROM golang:1.23-alpine AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
ARG APP_VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X main.version=${APP_VERSION}" \
    -o /out/claimsprocessor .

# ---------- Stage 3: minimal runtime image ----------
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
ENV PORT=8080 \
    STATIC_DIR=/app/web/dist
COPY --from=backend /out/claimsprocessor /app/claimsprocessor
COPY --from=frontend /app/web/dist /app/web/dist
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/claimsprocessor"]
