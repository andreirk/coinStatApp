# coinStatApp

## !!! Its not production ready : 
### --- Its uses float64 for currency calculations, has to be change to decimal library, or use fractional numbers.

A real-time crypto swap statistics system with high availability and minimal latency.

## Architecture
https://drive.google.com/file/d/185QYEC1IzOTNSMgeQ2zzEitjDjy73IkN/view?usp=sharing

This project follows Domain-Driven Design (DDD) and Clean Architecture principles:

- **Domain Layer**: Core business models, services, and repository interfaces
- **Application Layer**: Use cases and event processing
- **Infrastructure Layer**: Repository implementations (Redis, ClickHouse) and adapters
- **Handlers Layer**: HTTP API and WebSocket server (external interfaces)

### Dependency Inversion

The application uses dependency inversion to ensure that:
1. Domain logic depends only on abstractions
2. Infrastructure implementations depend on domain interfaces
3. Dependencies are injected at runtime in the application entry point

## Features

- Real-time swap statistics calculation (5min, 1h, 24h volumes and counts)
- HTTP API for querying statistics
- WebSocket for real-time updates
- High availability and restart recovery
- Event deduplication

## Configuration

Configuration is loaded from environment variables with sensible defaults.
You can also use a `.env` file for local development.

### Environment Variables

```
# Server
HTTP_PORT=8080

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# ClickHouse
CLICKHOUSE_DSN=localhost:9000

# Application
EVENT_BUFFER_SIZE=10000
DEBUG=false
```

## Getting Started

### Prerequisites

- Go 1.21+
- Redis (for caching and high-performance statistics)
- ClickHouse (for persistent storage and historical statistics)

### Running the Service

1. Clone the repository
2. Configure your environment variables (see `.env.example`)
3. Build and run:

```bash
go build -o coinStatApp ./cmd/coinStatApp
./coinStatApp
```

### API Endpoints

- `GET /stats` - Get all token statistics
- WebSocket `/ws` - Connect for real-time updates

## Testing

Run the tests with:

```bash
go test ./...
```

## Extending the Service

### Adding New Statistics

To add new statistics:

1. Extend the `Statistics` model in `internal/domain/model/statistics.go`
2. Update the StatisticsService implementation to calculate the new metrics
3. All consumers (HTTP/WebSocket) will automatically include the new statistics

### Adding New Storage Implementations

To add a new storage implementation:

1. Implement the `StatisticsPersistence` interface defined in `internal/domain/repository/interfaces.go`
2. Inject the new implementation in the application entry point (`cmd/coinStatApp/main.go`)

### Adding New Cache Implementations

To add a new caching implementation:

1. Implement the `StatisticsCache` interface defined in `internal/domain/repository/interfaces.go`
2. Inject the new implementation in the application entry point (`cmd/coinStatApp/main.go`)

## License

This project is licensed under the MIT License.
