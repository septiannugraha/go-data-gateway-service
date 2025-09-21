# API Test Commands for Go Data Gateway

A comprehensive list of curl commands to test all endpoints in the Go Data Gateway service.

## Prerequisites

Start the server with BigQuery credentials:
```bash
export GOOGLE_APPLICATION_CREDENTIALS="/home/septiannugraha/code/go-data-gateway/bigquery/spse-prod-sa-b00ef9fc8ed4.json"
export BIGQUERY_PROJECT_ID="gtp-data-prod"
export BIGQUERY_DATASET_ID=""
go run cmd/server/main_chi.go
```

## Health & Monitoring Endpoints

### Health Check
```bash
curl -X GET http://localhost:8081/health
```

### Readiness Check
Checks all data sources connectivity:
```bash
curl -X GET http://localhost:8081/ready
```

## BigQuery RUP Endpoints

### List RUP Data
Get paginated list of RUP KRO master data:
```bash
# Basic listing
curl -X GET "http://localhost:8081/api/v1/rup?limit=5" \
  -H "X-API-Key: demo-key-123" | jq

# With pagination
curl -X GET "http://localhost:8081/api/v1/rup?limit=10&offset=20" \
  -H "X-API-Key: demo-key-123" | jq
```

### Get RUP by ID
Get specific RUP by kd_kro_str:
```bash
curl -X GET "http://localhost:8081/api/v1/rup/EBA" \
  -H "X-API-Key: demo-key-123" | jq

# Another example
curl -X GET "http://localhost:8081/api/v1/rup/AEF" \
  -H "X-API-Key: demo-key-123" | jq
```

### Search RUP
Search with multiple filters:
```bash
# Search by keyword and year
curl -X POST http://localhost:8081/api/v1/rup/search \
  -H "Content-Type: application/json" \
  -H "X-API-Key: demo-key-123" \
  -d '{
    "keyword": "layanan",
    "tahun": "2023",
    "limit": 5
  }' | jq

# Search with budget range
curl -X POST http://localhost:8081/api/v1/rup/search \
  -H "Content-Type: application/json" \
  -H "X-API-Key: demo-key-123" \
  -d '{
    "tahun": "2023",
    "min_pagu": 1000000000,
    "max_pagu": 5000000000,
    "limit": 10
  }' | jq

# Search by satker
curl -X POST http://localhost:8081/api/v1/rup/search \
  -H "Content-Type: application/json" \
  -H "X-API-Key: demo-key-123" \
  -d '{
    "kd_satker": "181524",
    "limit": 5
  }' | jq
```

## Generic Query Endpoints

### BigQuery Direct Queries

```bash
# Simple test query
curl -X POST http://localhost:8081/api/v1/query \
  -H "Content-Type: application/json" \
  -H "X-API-Key: demo-key-123" \
  -d '{
    "source": "BIGQUERY",
    "sql": "SELECT current_timestamp() as timestamp, '\''BigQuery Works!'\'' as message"
  }' | jq

# Query RUP KRO Master directly
curl -X POST http://localhost:8081/api/v1/query \
  -H "Content-Type: application/json" \
  -H "X-API-Key: demo-key-123" \
  -d '{
    "source": "BIGQUERY",
    "sql": "SELECT * FROM `gtp-data-prod.layer_isb.rup_kromaster` WHERE tahun_anggaran = 2023 LIMIT 3"
  }' | jq

# Aggregation query (with LIMIT for cost safety)
curl -X POST http://localhost:8081/api/v1/query \
  -H "Content-Type: application/json" \
  -H "X-API-Key: demo-key-123" \
  -d '{
    "source": "BIGQUERY",
    "sql": "SELECT tahun_anggaran, COUNT(*) as total_kro, SUM(pagu_kro) as total_pagu FROM `gtp-data-prod.layer_isb.rup_kromaster` GROUP BY tahun_anggaran ORDER BY tahun_anggaran DESC LIMIT 5"
  }' | jq

# Get unique KLPD list
curl -X POST http://localhost:8081/api/v1/query \
  -H "Content-Type: application/json" \
  -H "X-API-Key: demo-key-123" \
  -d '{
    "source": "BIGQUERY",
    "sql": "SELECT DISTINCT kd_klpd, nama_klpd, jenis_klpd FROM `gtp-data-prod.layer_isb.rup_kromaster` ORDER BY nama_klpd LIMIT 20"
  }' | jq
```

### Dremio/Iceberg Queries

```bash
# Query tender data from Dremio
curl -X POST http://localhost:8081/api/v1/query \
  -H "Content-Type: application/json" \
  -H "X-API-Key: demo-key-123" \
  -d '{
    "source": "DATAWAREHOUSE",
    "sql": "SELECT * FROM nessie_iceberg.tender_data LIMIT 5"
  }' | jq

# Aggregation on Iceberg
curl -X POST http://localhost:8081/api/v1/query \
  -H "Content-Type: application/json" \
  -H "X-API-Key: demo-key-123" \
  -d '{
    "source": "DATAWAREHOUSE",
    "sql": "SELECT tahun_anggaran, COUNT(*) as total_tender FROM nessie_iceberg.tender_data GROUP BY tahun_anggaran ORDER BY tahun_anggaran DESC LIMIT 10"
  }' | jq
```

## Dremio/Iceberg Tender Endpoints

### List Tenders
```bash
# Basic listing
curl -X GET "http://localhost:8081/api/v1/tender?page=1&limit=10" \
  -H "X-API-Key: demo-key-123" | jq

# Different page
curl -X GET "http://localhost:8081/api/v1/tender?page=2&limit=20" \
  -H "X-API-Key: demo-key-123" | jq
```

### Get Tender by ID
```bash
curl -X GET "http://localhost:8081/api/v1/tender/80210078585001" \
  -H "X-API-Key: demo-key-123" | jq
```

### Search Tenders
```bash
# Search by keyword
curl -X POST http://localhost:8081/api/v1/tender/search \
  -H "Content-Type: application/json" \
  -H "X-API-Key: demo-key-123" \
  -d '{
    "keyword": "infrastruktur",
    "limit": 5
  }' | jq

# Search with multiple filters
curl -X POST http://localhost:8081/api/v1/tender/search \
  -H "Content-Type: application/json" \
  -H "X-API-Key: demo-key-123" \
  -d '{
    "keyword": "pembangunan",
    "tahun_anggaran": "2024",
    "status": "Selesai",
    "min_nilai": 1000000000,
    "max_nilai": 50000000000,
    "provinsi": "DKI Jakarta",
    "limit": 10
  }' | jq
```

## Performance Testing

### Response Time Test
```bash
# Test BigQuery response time
time curl -X POST http://localhost:8081/api/v1/query \
  -H "Content-Type: application/json" \
  -H "X-API-Key: demo-key-123" \
  -d '{
    "source": "BIGQUERY",
    "sql": "SELECT COUNT(*) as total FROM `gtp-data-prod.layer_isb.rup_kromaster`"
  }' -o /dev/null -s

# Test with detailed metrics
curl -X GET "http://localhost:8081/api/v1/rup?limit=100" \
  -H "X-API-Key: demo-key-123" \
  -o /dev/null -s -w "HTTP Status: %{http_code}\nTotal Time: %{time_total}s\nConnect Time: %{time_connect}s\nStart Transfer: %{time_starttransfer}s\nSize: %{size_download} bytes\n"
```

### Load Testing with hey
```bash
# Install hey first: go install github.com/rakyll/hey@latest

# Test RUP endpoint
hey -n 100 -c 10 -H "X-API-Key: demo-key-123" \
  http://localhost:8081/api/v1/rup?limit=10

# Test query endpoint
hey -n 50 -c 5 -m POST \
  -H "Content-Type: application/json" \
  -H "X-API-Key: demo-key-123" \
  -d '{"source":"BIGQUERY","sql":"SELECT COUNT(*) FROM `gtp-data-prod.layer_isb.rup_kromaster`"}' \
  http://localhost:8081/api/v1/query
```

## Error Handling Tests

### Authentication Errors
```bash
# Wrong API key (should return 401)
curl -X GET "http://localhost:8081/api/v1/rup?limit=5" \
  -H "X-API-Key: wrong-key" -w "\nHTTP Status: %{http_code}\n"

# Missing API key (should return 401)
curl -X GET "http://localhost:8081/api/v1/rup?limit=5" \
  -w "\nHTTP Status: %{http_code}\n"
```

### Forbidden Operations
```bash
# Try DELETE operation (should be blocked)
curl -X POST http://localhost:8081/api/v1/query \
  -H "Content-Type: application/json" \
  -H "X-API-Key: demo-key-123" \
  -d '{
    "source": "BIGQUERY",
    "sql": "DELETE FROM `gtp-data-prod.layer_isb.rup_kromaster` WHERE 1=1"
  }' | jq

# Try INSERT operation (should be blocked)
curl -X POST http://localhost:8081/api/v1/query \
  -H "Content-Type: application/json" \
  -H "X-API-Key: demo-key-123" \
  -d '{
    "source": "BIGQUERY",
    "sql": "INSERT INTO `gtp-data-prod.layer_isb.rup_kromaster` VALUES (1, '\''test'\'')"
  }' | jq
```

### Invalid Requests
```bash
# Invalid source
curl -X POST http://localhost:8081/api/v1/query \
  -H "Content-Type: application/json" \
  -H "X-API-Key: demo-key-123" \
  -d '{
    "source": "INVALID_SOURCE",
    "sql": "SELECT 1"
  }' | jq

# Malformed JSON
curl -X POST http://localhost:8081/api/v1/rup/search \
  -H "Content-Type: application/json" \
  -H "X-API-Key: demo-key-123" \
  -d '{"keyword": "test"' | jq
```

## Swagger UI

Open the API documentation in your browser:
```bash
# macOS
open http://localhost:8081/api/swagger-ui.html

# Linux
xdg-open http://localhost:8081/api/swagger-ui.html

# Or just navigate to:
# http://localhost:8081/api/swagger-ui.html
```

## Important Notes

1. **Cost Safety**: All queries include `LIMIT` clauses to prevent expensive BigQuery scans
2. **Authentication**: Use `X-API-Key: demo-key-123` for testing
3. **Data Sources**:
   - `BIGQUERY` - For BigQuery queries (RUP data)
   - `DATAWAREHOUSE` - For Dremio/Iceberg queries (Tender data)
4. **Response Format**: All responses follow the standard format with `success`, `data`, and optional `meta` fields
5. **Error Handling**: Errors return `success: false` with error details

## Troubleshooting

If endpoints fail:
1. Check if server is running on port 8081
2. Verify environment variables are set (GOOGLE_APPLICATION_CREDENTIALS, BIGQUERY_PROJECT_ID)
3. Check if Dremio is running on port 32010 (for Arrow Flight SQL)
4. Ensure BigQuery service account has proper permissions
5. Check server logs for detailed error messages