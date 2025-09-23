# Inaproc API Gateway

> **Indonesia Procurement Data Gateway** - A comprehensive API solution for secure access to Indonesia's procurement data, managed by LKPP (Lembaga Kebijakan Pengadaan Barang/Jasa Pemerintah).

## 🌟 Overview

The Inaproc API Gateway is a multi-component system designed to provide secure, scalable, and efficient access to Indonesia's procurement data for government partners. The architecture combines modern API management with high-performance data services to deliver a robust procurement data platform.

## 🏗️ Architecture

```
┌─────────────────────┐    ┌─────────────────────┐    ┌─────────────────────┐
│                     │    │                     │    │                     │
│  Government         │    │  Inaproc API        │    │  Data Gateway       │
│  Partners           │────│  Gateway            │────│  Service            │
│  (External)         │    │  (Fusio)            │    │  (Go)               │
│                     │    │                     │    │                     │
└─────────────────────┘    └─────────────────────┘    └─────────────────────┘
                                      │                          │
                                      │                          │
                                      ▼                          ▼
                           ┌─────────────────────┐    ┌─────────────────────┐
                           │                     │    │                     │
                           │  Documentation      │    │  Data Sources       │
                           │  Portal             │    │  • Iceberg + Nessie │
                           │  (Fumadocs)         │    │  • PostgreSQL       │
                           │                     │    │  • Cloud Storage    │
                           └─────────────────────┘    └─────────────────────┘
```

## 📁 Project Structure

```
├── inaproc-api-gateway/     # Public-facing API Gateway (Fusio-based)
│   └── (to be implemented)  # Authentication, rate limiting, API analytics
├── data-gateway-service/    # Data orchestration service (Go-based)
│   ├── cmd/                 # Application entry points
│   ├── internal/            # Business logic and domain models
│   ├── pkg/                 # Shared utilities
│   └── configs/             # Configuration files
├── docs/                    # API documentation (Fumadocs-based)
│   ├── content/             # Documentation content
│   ├── src/                 # Documentation UI
│   └── scripts/             # Documentation generation scripts
└── prd.md                   # Technical architecture documentation
```

## 🔧 Components

### 1. Inaproc API Gateway (`inaproc-api-gateway/`)

**Technology**: [Fusio](https://github.com/apioo/fusio) (PHP-based API Management Platform)

**Responsibilities**:
- 🔐 **Authentication & Authorization**: Dual-token system (long-term + short-term)
- 🚦 **Rate Limiting**: Per-partner request throttling
- 📊 **Analytics**: API usage monitoring and reporting
- 🛡️ **Security**: Request validation and partner management
- 🔄 **API Routing**: Request forwarding to data gateway service

**Key Features**:
- RESTful API endpoints for procurement data
- Partner credential management
- Request/response logging and auditing
- OpenAPI specification generation

### 2. Data Gateway Service (`data-gateway-service/`)

**Technology**: Go 1.21+ with Chi router framework

**Responsibilities**:
- 🗄️ **Multi-source Data Integration**: Orchestrates queries across different data sources
- 🧠 **Intelligent Query Routing**: Determines optimal data source based on query characteristics
- ⚡ **Performance Optimization**: Caching, query optimization, and connection pooling
- 🔄 **Data Transformation**: Standardizes data formats from various sources
- 🔒 **Internal Security**: mTLS and API key authentication with API Gateway

**Data Sources**:
- **Primary**: Iceberg + Nessie Data Warehouse (analytical data)
- **Secondary**: CloudSQL PostgreSQL (operational data)
- **Additional**: Cloud Storage, Legacy databases, External APIs

### 3. Documentation Portal (`docs/`)

**Technology**: [Fumadocs](https://fumadocs.dev/) (Next.js-based documentation framework)

**Responsibilities**:
- 📚 **API Documentation**: Interactive API reference
- 🔍 **OpenAPI Integration**: Auto-generated API specs from Fusio
- 👨‍💻 **Developer Guides**: Integration tutorials and examples
- 🎨 **Interactive UI**: Modern, searchable documentation interface

## 🚀 Quick Start

### Prerequisites

- **Go** 1.21+ (for data gateway service)
- **PHP** 8.2+ (for Fusio API gateway)
- **Node.js** 20+ (for documentation)
- **Docker** & **Docker Compose** (recommended for local development)
- **PostgreSQL** 14+ (for operational data)

### Development Setup

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd go-data-gateway-service
   ```

2. **Setup Data Gateway Service**:
   ```bash
   cd data-gateway-service
   cp configs/config.example.yaml configs/config.yaml
   go mod download
   go run cmd/server/main.go
   ```

3. **Setup Documentation**:
   ```bash
   cd docs
   npm install
   npm run dev
   ```

4. **Setup API Gateway** (coming soon):
   ```bash
   cd inaproc-api-gateway
   # Setup instructions will be added when implemented
   ```

### Docker Development

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

## 📊 Available APIs

### Procurement Data Endpoints

- **Tender Data**: `/v1/tender` - Active and historical tender information
- **RUP Data**: `/v1/rup` - Procurement planning (Rencana Umum Pengadaan)
- **Katalog V6**: `/v1/katalog` - Government product/service catalog

### Authentication Flow

1. **Token Exchange**: Exchange long-term token for short-term access token
2. **API Access**: Use short-term token to access procurement data
3. **Token Refresh**: Automatically refresh tokens before expiration

## 🔐 Security Features

- **Dual Authentication**: Long-term + short-term token system
- **mTLS Communication**: Secure service-to-service communication
- **Rate Limiting**: Configurable per-partner limits
- **Audit Logging**: Comprehensive request tracking
- **Data Encryption**: At rest and in transit

## 🗄️ Data Architecture

- **Data Warehouse**: Apache Iceberg with Nessie catalog for analytical queries
- **Operational Database**: PostgreSQL for real-time data and metadata
- **Query Engine**: Trino/Presto for distributed query processing
- **Caching Layer**: Redis for performance optimization

## 📈 Monitoring & Analytics

- **Metrics**: Prometheus-compatible metrics collection
- **Logging**: Structured logging with correlation IDs
- **Tracing**: Distributed tracing for request flow analysis
- **Health Checks**: Comprehensive service health monitoring

## 🤝 Contributing

1. Read the [Technical Architecture Documentation](./prd.md)
2. Follow the coding standards for each component
3. Add tests for new features
4. Submit pull requests with clear descriptions

## 📚 Documentation

- **Technical Architecture**: [prd.md](./prd.md) - Comprehensive technical documentation
- **API Reference**: `/docs` - Interactive API documentation
- **Developer Portal**: Live documentation with examples and tutorials

## 📄 License

This project is developed for LKPP Indonesia as part of the Indonesia Procurement Data Gateway initiative.

## 🆘 Support

For technical support and questions:
- Create an issue in this repository
- Contact the LKPP technical team
- Refer to the comprehensive documentation in [prd.md](./prd.md)

---

**Built with ❤️ for Indonesia's Procurement Transparency Initiative**