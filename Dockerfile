# ---------- Tahap build ----------
FROM golang:1.26-alpine AS build
WORKDIR /src

# Download dependency dulu (di-cache kalau go.mod tidak berubah)
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/server ./cmd/server

# ---------- Tahap jalan ----------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/server ./server
COPY web ./web
USER nobody
EXPOSE 8080
CMD ["./server"]
