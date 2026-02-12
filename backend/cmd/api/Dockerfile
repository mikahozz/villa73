# base image
FROM golang:alpine3.20 as base
WORKDIR /builder
ENV CGO_ENABLED=0

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the whole project
COPY . ./

# Build the application
RUN go build -o /builder/main ./cmd/api

# runner image
FROM gcr.io/distroless/static-debian11:nonroot
WORKDIR /app
COPY --from=base /builder/main /builder/.env ./

EXPOSE 6001
CMD ["/app/main"]