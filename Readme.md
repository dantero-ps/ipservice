# IP Geolocation Service

A high-performance service that provides IP to country mapping using data from Regional Internet Registries (RIRs).

## Features

- Aggregates IP range data from all 5 RIRs (ARIN, RIPE, APNIC, LACNIC, AFRINIC)
- Supports both IPv4 and IPv6 addresses
- Multi-level caching with Redis
- PostgreSQL for persistent storage
- RESTful API endpoint for IP lookups
- Automatic daily updates of IP ranges
- Efficient request sampling for monitoring
- Production-ready error handling and logging

## Prerequisites

- Go 1.21 or higher
- PostgreSQL 15 or higher
- Redis 7 or higher
- Docker and Docker Compose (optional)

## Quick Start

1. Clone the repository:
```bash
git clone https://github.com/dantero-ps/ipservice.git
cd ipservice
```

2. Start the services using Docker Compose:
```bash
docker-compose up -d
```

3. The service will be available at `http://localhost:8080`

## API Usage

### Lookup IP Address

```bash
curl http://localhost:8080/api/v1/lookup/8.8.8.8
```

Response:
```json
{
    "ip": "8.8.8.8",
    "country_code": "US"
}
```

### Health Check

```bash
curl http://localhost:8080/api/v1/health
```

Response:
```json
{
    "status": "healthy"
}
```

## Configuration

Environment variables:
- `POSTGRES_URL`: PostgreSQL connection string (default: "postgres://postgres:postgres@localhost:5432/ipservice?sslmode=disable")
- `REDIS_URL`: Redis connection string (default: "redis://localhost:6379/0")
- `SERVER_PORT`: HTTP server port (default: ":8080")

## Development

1. Install dependencies:
```bash
go mod download
```

2. Run tests:
```bash
go test ./...
```

3. Build:
```bash
go build -o ipservice ./cmd/ipservice
```

## Performance

- Handles thousands of requests per second
- Request sampling logs 0.1% of successful requests
- All errors and slow requests (>100ms) are logged
- Multi-level caching strategy:
  - Direct IP cache in Redis
  - IP range cache in Redis
  - PostgreSQL for persistent storage

## Data Sources

- ARIN: https://ftp.arin.net/pub/stats/arin/delegated-arin-extended-latest
- RIPE: https://ftp.ripe.net/pub/stats/ripencc/delegated-ripencc-latest
- APNIC: https://ftp.apnic.net/stats/apnic/delegated-apnic-latest
- LACNIC: https://ftp.lacnic.net/pub/stats/lacnic/delegated-lacnic-latest
- AFRINIC: https://ftp.afrinic.net/stats/afrinic/delegated-afrinic-latest

## License

MIT License

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
