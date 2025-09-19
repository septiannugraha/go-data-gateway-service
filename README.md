GO DATA GATEWAY SERVICE

Production-ready Go service for querying Dremio/Iceberg and BigQuery with comprehensive logging, caching, and monitoring.

## Quick Start

1. Clone and setup:
```bash
git clone <repo>
cd go-data-gateway
cp .env.example .env
# Edit .env with your credentials
```

2. Run with Docker:
```bash
docker-compose up
```

3. Test endpoints:
```bash
# Health check
curl http://localhost:8080/health

# Tender data from Dremio/Iceberg
curl -H "X-API-Key: demo-key-123" http://localhost:8080/api/v1/tender

# RUP data from BigQuery
curl -H "X-API-Key: demo-key-123" http://localhost:8080/api/v1/rup
```

## Architecture

```
Internet → Fusio (Port 80)
            ↓
         Go Gateway (Port 8080)
            ├── /tender → Dremio/Iceberg
            ├── /rup → BigQuery
            └── Redis Cache
```

## API Endpoints

### Authentication
All endpoints require API key in header:
```
X-API-Key: your-api-key
```

### Tender Endpoints (Dremio/Iceberg)

**List Tenders**
```
GET /api/v1/tender?limit=100&offset=0&status=active
```

**Get Tender by ID**
```
GET /api/v1/tender/{id}
```

**Search Tenders**
```
POST /api/v1/tender/search
{
  "keyword": "construction",
  "min_value": 1000000,
  "max_value": 10000000,
  "status": ["active", "closed"],
  "tahun_anggaran": 2024
}
```

### RUP Endpoints (BigQuery)

**List RUP**
```
GET /api/v1/rup?limit=100&offset=0
```

**Search RUP**
```
POST /api/v1/rup/search
{
  "keyword": "pengadaan",
  "year": 2024
}
```

### Generic Query Endpoint

**Execute Custom Query**
```
POST /api/v1/query
{
  "source": "dremio",  // or "bigquery"
  "sql": "SELECT * FROM table LIMIT 10"
}
```

## Development

### Without Docker
```bash
# Install dependencies
go mod download

# Run locally
go run cmd/server/main.go
```

### With Live Reload
```bash
# Install air
go install github.com/cosmtrek/air@latest

# Run with hot reload
air
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| PORT | Server port | 8080 |
| ENV | Environment (development/production) | development |
| API_KEYS | Comma-separated API keys | demo-key-123 |
| RATE_LIMIT | Requests per minute | 100 |
| DREMIO_HOST | Dremio server host | - |
| DREMIO_PORT | Dremio server port | 31010 |
| BIGQUERY_PROJECT_ID | GCP project ID | - |
| REDIS_HOST | Redis host | localhost |

### BigQuery Setup

1. Create service account in GCP Console
2. Download JSON key file
3. Place in `credentials/bigquery-key.json`
4. Set `GOOGLE_APPLICATION_CREDENTIALS` in .env

### Dremio Setup

Option 1: Username/Password
```env
DREMIO_USERNAME=your-username
DREMIO_PASSWORD=your-password
```

Option 2: Token
```env
DREMIO_TOKEN=your-token
```

## AI Code Generation

Use this prompt to generate new endpoints:

```
Create a new endpoint /api/v1/procurement that:
1. Queries the iceberg.procurement_2024 table
2. Accepts filters: status, date_range, min_value, max_value
3. Returns paginated results with total count
4. Includes comprehensive logging
5. Caches results for 5 minutes
```

## Monitoring

### Prometheus Metrics
Available at http://localhost:9090

### Grafana Dashboards
Access at http://localhost:3000 (admin/admin)

### Logs
Structured JSON logs with Zap:
```json
{
  "level": "info",
  "ts": "2024-01-15T10:30:00Z",
  "msg": "Query executed",
  "sql": "SELECT * FROM tender",
  "duration": "150ms",
  "rows": 100
}
```

## Performance

- Redis caching: 5-minute TTL
- Connection pooling for Dremio/BigQuery
- Graceful shutdown
- Rate limiting per API key
- Response time <200ms for cached queries

## Security

- API key authentication
- SQL injection prevention (read-only queries)
- TLS ready (configure in Fusio)
- No credentials in code

## Deployment

### Production with TLS
```bash
# Edit docker-compose.yml to add certificates
docker-compose -f docker-compose.prod.yml up -d
```

### Kubernetes
```bash
kubectl apply -f k8s/deployment.yaml
```

### Scaling
- Horizontal scaling: Run multiple Go service instances
- Cache scaling: Use Redis Cluster
- Database scaling: Dremio/BigQuery handle their own scaling

## Testing

```bash
# Run tests
go test ./...

# With coverage
go test -cover ./...

# Load testing
hey -n 10000 -c 100 -H "X-API-Key: demo-key-123" http://localhost:8080/api/v1/tender
```

## Troubleshooting

### Dremio Connection Failed
- Check DREMIO_HOST and DREMIO_PORT
- Verify credentials
- Test with: `curl http://dremio-host:9047/apiv2/login`

### BigQuery Permission Denied
- Check service account permissions
- Verify GOOGLE_APPLICATION_CREDENTIALS path
- Test with: `gcloud auth application-default login`

### High Memory Usage
- Reduce cache TTL
- Implement query result pagination
- Check for memory leaks with pprof

## License

MIT