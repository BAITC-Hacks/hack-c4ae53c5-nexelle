FROM golang:1.27-alpine AS build

WORKDIR /src
COPY go.mod ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /career-quest ./cmd/career-quest

FROM alpine:3.21

WORKDIR /app
COPY --from=build /career-quest /app/career-quest

ENV APP_PORT=8080
ENV DATA_DIR=/app/data
ENV SNAPSHOT_DATE=2026-10-01

EXPOSE 8080
ENTRYPOINT ["/app/career-quest"]
