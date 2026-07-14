FROM golang:1.25-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG APP_VERSION=dev
ARG GIT_COMMIT=unknown
RUN CGO_ENABLED=0 GOOS=linux go build \
  -ldflags="-X chateauneuf-portaria-backend/internal/version.Version=${APP_VERSION} -X chateauneuf-portaria-backend/internal/version.Commit=${GIT_COMMIT}" \
  -o /out/chateauneuf-portaria-api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=build /out/chateauneuf-portaria-api /app/chateauneuf-portaria-api
COPY migrations /app/migrations

EXPOSE 8080

ENTRYPOINT ["/app/chateauneuf-portaria-api"]
