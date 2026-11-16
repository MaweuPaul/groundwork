# One image for the Go API and the Python worker, so both fit on a single free web service.
# Build from backend/:  docker build -t groundwork-backend backend

FROM golang:1.26-alpine AS go-build
WORKDIR /src
COPY go-api/go.mod go-api/go.sum ./
RUN go mod download
COPY go-api/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM python:3.12-slim
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1
WORKDIR /app

COPY gee-service/requirements.txt worker/requirements.txt
RUN pip install --no-cache-dir -r worker/requirements.txt

COPY gee-service/app worker/app
COPY go-api/migrations migrations
COPY --from=go-build /out/api /out/migrate bin/
COPY start.sh start.sh
RUN chmod +x start.sh && useradd --system --no-create-home groundwork
USER groundwork

CMD ["./start.sh"]
