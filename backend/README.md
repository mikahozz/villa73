# GoHome

## Work

### Next

- [x] Change the sync to update the returned prices into db
- [ ] Fix the prices to correct ones. Entsoe returns weird data
- [ ] Change prices api to read data from db

## Testing

This project contains both unit tests and integration tests. Integration tests require a running PostgreSQL database (provided via Docker).

### Prerequisites

- Go 1.21 or newer
- Docker and Docker Compose
- Make sure `.env` file exists (copy from `.env.example` if needed)
- Start the database:

```
docker compose up -d db
```

### Running Tests

#### All Tests

- Run all tests including integration tests

```
go test -v ./...
```

- Run integration tests for specific package

```
go test -v ./dbsync
```

#### Unit Tests Only

```
go test -v -short ./...
```
