# Inaproc API Gateway - Technical Architecture Documentation

## 📋 Table of Contents

* [Overview](#-overview)
* [System Architecture](#-system-architecture)
* [Components](#-components)
* [Data Catalog](#-data-catalog)
* [API Specifications](#-api-specifications)
* [Authentication & Security](#-authentication--security)
* [Deployment](#-deployment)
* [Development Guide](#-development-guide)


## 🌟 Overview

The **Inaproc API Gateway** is a comprehensive solution designed to provide secure and efficient access to Indonesia's procurement data for government partners. Built with modern architecture principles, it combines the power of [Fusio](https://github.com/apioo/fusio) as the API gateway layer with a custom Go-based data service for optimal performance and scalability.

### Key Features

* 🔐 **Dual Authentication Flow** - Long-term + Short-term token system
* 🚀 **High Performance** - Go-based data service with Chi framework
* 🗄️ **Modern Data Stack** - Iceberg + Nessie data warehouse
* 🔄 **Multi-Source Integration** - Support for various data sources
* 📊 **Real-time Analytics** - Built-in monitoring and analytics
* 🛡️ **Enterprise Security** - mTLS and API key protection


---

## 🏗️ System Architecture

### High-Level Architecture

 ![](/api/attachments.redirect?id=9d7fa07f-be69-480e-b6bb-ffc44b9f1c4d " =354x354")

### Complete Request Journey: From Partner to Data

Understanding the complete flow of a procurement data request is crucial to grasp how the Inaproc API Gateway system operates. Let's walk through the entire journey from when a government partner needs procurement data to when they receive the response.

#### 🚀 **Step 1: Initial Authentication Setup**

Government partners begin their integration journey by receiving **long-term tokens** through secure channels (encrypted email or secure file transfer):

```
Partner receives:
- Long-term token: "ltt_ministry_pu_2024_abc123def456..."
- Client ID: "partner_ministry_pu"
- Base API URL: "https://api.inaproc.lkpp.go.id/v1"
```

#### 🔐 **Step 2: Token Exchange Flow**

Before accessing any procurement data, partners must exchange their long-term token for a short-term access token:

**Request to API Gateway:**

```http

POST https://api.inaproc.lkpp.go.id/v1/auth/token

Content-Type: application/json

{
  "long_term_token": "ltt_ministry_pu_2024_abc123def456...",
  "client_id": "partner_ministry_pu"
}
```

**API Gateway Processing:**


1. **Validates** the long-term token against its internal database
2. **Checks** partner permissions and access levels
3. **Generates** a short-term JWT token (expires in 1 hour)
4. **Logs** the authentication event for audit purposes

**Response from API Gateway:**

```http

HTTP/1.1 200 OK

Content-Type: application/json

{
  "short_term_token": "stt_xyz789abc123...",
  "expires_in": 3600,
  "token_type": "Bearer",
  "scope": "tender.read rup.read katalog.read"
}
```

#### 📊 **Step 3: Data Request Initiation**

Now equipped with a valid short-term token, the partner can request procurement data:

**Partner's Data Request:**

```http

GET https://api.inaproc.lkpp.go.id/v1/tender?klpd_id=66&month=12&year=2024&status=ACTIVE&page=1&limit=20

Authorization: Bearer stt_xyz789abc123...
X-API-Client-ID: partner_ministry_pu

Accept: application/json
```

#### 🛡️ **Step 4: API Gateway Security & Validation**

The **Inaproc API Gateway** (powered by Fusio) performs comprehensive security checks:


1. **Token Validation:**
   * Verifies JWT signature and expiration
   * Checks token scope against requested endpoint
   * Validates partner permissions for the specific KLPD data
2. **Request Validation:**
   * Sanitizes query parameters (`klpd_id=66`, `month=12`, etc.)
   * Validates parameter types and ranges
   * Applies rate limiting rules (e.g., 100 requests/minute)
3. **Audit Logging:**
   * Records request details (partner, endpoint, parameters, timestamp)
   * Tracks API usage for billing and monitoring

#### 🔄 **Step 5: Secure Communication to Data Gateway Service**

Once validated, the API Gateway forwards the request to the **Data Gateway Service** with enhanced security:

**Internal Request (API Gateway → Data Gateway Service):**

```http

GET http://data-gateway-service:8080/internal/tender?klpd_id=66&month=12&year=2024&status=ACTIVE&page=1&limit=20

X-API-Key: static_secure_api_key_abc123xyz789

X-Request-ID: req_abc123def456

X-Partner-ID: partner_ministry_pu

X-Original-IP: 203.194.112.45

Content-Type: application/json
```

**Security Layers:**

* **Static API Key:** Shared secret between API Gateway and Data Gateway Service
* **mTLS (Mutual TLS):** Certificate-based authentication at infrastructure level
* **Network Isolation:** Services communicate within secure private network
* **Request Context:** Forward partner information for proper data filtering

#### 🧠 **Step 6: Data Gateway Service Intelligence**

The **Data Gateway Service** (Go-based) receives the request and makes intelligent decisions:


1. **Query Analysis:**

   ```go
   // Analyze request characteristics
   queryType := analyzer.AnalyzeQuery(params)
   // Result: "HISTORICAL_TENDER_DATA_WITH_KLPD_FILTER"
   ```
2. **Data Source Selection:**
   * **Primary Source:** Iceberg Data Warehouse (for historical tender data)
   * **Secondary Source:** CloudSQL PostgreSQL (for real-time status updates)
   * **Enrichment Sources:** Document storage, KLPD master data
3. **Query Optimization:**

   ```go
   optimizedQuery := optimizer.BuildQuery(QueryParams{
       Entity: "tender",
       Filters: map[string]interface{}{
           "klpd_id": 66,
           "announcement_month": 12,
           "announcement_year": 2024,
           "status": "ACTIVE",
       },
       Pagination: PaginationParams{Page: 1, Limit: 20},
   })
   ```

#### 🗄️ **Step 7: Multi-Source Data Orchestration**

The Data Gateway Service orchestrates data retrieval from multiple sources:

**Primary Query to Data Warehouse (Iceberg + Nessie):**

```sql
-- Executed via Trino/Presto query engine

SELECT 
    t.tender_id,
    t.tender_name,
    t.budget_amount,
    t.announcement_date,
    t.closing_date,
    t.procurement_method,
    k.klpd_name

FROM iceberg.procurement.tender_fact t

INNER JOIN iceberg.procurement.klpd_dim k ON t.klpd_id = k.klpd_id

WHERE t.klpd_id = 66
  AND t.partition_year = 2024
  AND t.partition_month = 12
  AND t.status = 'ACTIVE'
ORDER BY t.announcement_date DESC

LIMIT 20 OFFSET 0;
```

**Enrichment Query to PostgreSQL:**

```sql
-- Get real-time status updates and additional metadata

SELECT 
    tender_id,
    current_status,
    last_updated,
    document_count,
    bidder_count

FROM tender_realtime_status 
WHERE tender_id IN ($1, $2, $3, ...);
```

**Document Metadata Query:**

```sql
-- Get document information from cloud storage metadata

SELECT 
    entity_id as tender_id,
    document_type,
    document_name,
    storage_url,
    file_size

FROM document_metadata 
WHERE entity_type = 'tender' 
  AND entity_id IN ($1, $2, $3, ...);
```

#### 🔧 **Step 8: Data Processing & Transformation**

The Data Gateway Service processes and combines the data:


1. **Data Merging:**

   ```go
   // Combine data from multiple sources
   enrichedTenders := dataProcessor.MergeDataSources(
       dwhResults,      // Primary tender data
       realtimeStatus,  // Current status updates
       documentMeta,    // Document information
   )
   ```
2. **Format Standardization:**

   ```go
   // Convert to standard API response format
   response := formatter.FormatTenderResponse(enrichedTenders, pagination)
   ```
3. **Data Validation:**

   ```go
   // Ensure data quality and completeness
   validatedResponse := validator.ValidateResponse(response)
   ```

#### 📤 **Step 9: Response Assembly & Caching**

The Data Gateway Service prepares the final response:


1. **Response Formatting:**

   ```json
   {
     "status": "success",
     "data": {
       "items": [
         {
           "tender_id": "TND-2024-001234",
           "klpd_id": 66,
           "klpd_name": "Kementerian Pekerjaan Umum dan Perumahan Rakyat",
           "tender_name": "Pembangunan Jalan Tol Jakarta-Surabaya Seksi 3",
           "budget_amount": 15000000000.00,
           "status": "ACTIVE",
           "document_count": 5
         }
       ],
       "pagination": {
         "current_page": 1,
         "total_pages": 45,
         "total_items": 892
       }
     }
   }
   ```
2. **Intelligent Caching:**

   ```go
   // Cache strategy based on data characteristics
   if shouldCache(queryParams) {
       cacheManager.Set(cacheKey, response, TTL(30*time.Minute))
   }
   ```

#### 🔙 **Step 10: Response Journey Back to Partner**

The response travels back through the secure chain:

**Data Gateway Service → API Gateway:**

```http

HTTP/1.1 200 OK

Content-Type: application/json

X-Request-ID: req_abc123def456

X-Query-Time: 245ms

X-Data-Sources: dwh,postgresql,cache

{
  "status": "success",
  "data": { ... },
  "meta": {
    "query_time": "245ms",
    "data_sources": ["dwh", "postgresql"],
    "cache_hit": false
  }
}
```

**API Gateway → Government Partner:**

```http

HTTP/1.1 200 OK

Content-Type: application/json

X-Request-ID: req_abc123def456

X-Rate-Limit-Remaining: 85

X-Rate-Limit-Reset: 1703847600

{
  "status": "success",
  "data": { ... },
  "meta": {
    "timestamp": "2024-12-19T10:30:00Z",
    "request_id": "req_abc123def456"
  }
}
```

#### 📊 **Step 11: Monitoring & Analytics**

Throughout this entire journey, comprehensive monitoring occurs:

**API Gateway Metrics:**

* Request/response times, error rates
* Partner usage patterns and quota consumption
* Authentication success/failure rates

**Data Gateway Service Metrics:**

* Query performance across different data sources
* Cache hit/miss ratios
* Data freshness and quality metrics

**Infrastructure Metrics:**

* Network latency between services
* Database connection pool utilization
* Resource usage (CPU, memory, storage)

#### 🔄 **Performance Optimizations in the Flow**

**Caching Strategy:**

* **L1 (API Gateway):** Response caching for identical requests
* **L2 (Data Gateway):** Query result caching with smart invalidation
* **L3 (Data Sources):** Database query plan caching

**Parallel Processing:**

* Data Gateway Service can query multiple sources simultaneously
* Background data enrichment for commonly requested information
* Prefetching of frequently accessed reference data

**Intelligent Routing:**

* Recent data requests routed to operational databases
* Historical analysis requests routed to data warehouse
* Cached responses served for identical recent queries

This complete journey showcases how the Inaproc API Gateway system seamlessly orchestrates secure, high-performance access to Indonesia's procurement data while maintaining strict security, comprehensive monitoring, and optimal performance at every step.


### Why Inaproc API Gateway is Essential

The **Inaproc API Gateway** serves as the critical public-facing entry point for Indonesia's procurement data ecosystem. Here's why this component is indispensable:

#### 🎯 **Centralized Access Control**

* **Single Point of Entry:** Government partners access all procurement data through one unified interface, eliminating the complexity of connecting to multiple disparate systems
* **Standardized Authentication:** Implements a robust dual-token authentication system that ensures secure access while maintaining ease of use for partners
* **Partner Management:** Centralized management of partner credentials, permissions, and access levels with comprehensive audit trails

#### 🛡️ **Security & Compliance**

* **Data Protection:** Acts as a security perimeter that protects sensitive procurement data from unauthorized access
* **Regulatory Compliance:** Ensures all data access complies with Indonesian government data protection and transparency regulations
* **Threat Mitigation:** Provides DDoS protection, SQL injection prevention, and other security measures to protect the underlying infrastructure

#### 📊 **Operational Excellence**

* **Rate Limiting:** Prevents system overload by implementing intelligent rate limiting per partner and endpoint
* **Analytics & Monitoring:** Comprehensive logging and analytics to understand API usage patterns, identify issues, and optimize performance
* **API Versioning:** Manages multiple API versions to ensure backward compatibility while enabling continuous improvement

#### 🚀 **Scalability & Performance**

* **Load Distribution:** Distributes incoming requests efficiently across backend services
* **Caching Strategy:** Implements intelligent caching to reduce response times and backend load
* **Geographic Distribution:** Can be deployed across multiple regions for optimal performance

### Data Gateway Service: The Intelligent Data Bridge

The **Data Gateway Service** is the sophisticated middleware that bridges the gap between the API Gateway and the complex data landscape of LKPP. This Go-based service is architected to handle the intricacies of modern data integration.

#### 🔗 **Multi-Source Data Integration**

**Primary Data Sources:**

* **Iceberg + Nessie Data Warehouse:** The main analytical data store containing historical and processed procurement data
* **CloudSQL PostgreSQL:** Operational database for real-time data and service metadata
* **Legacy SQL Servers:** Integration with existing government database systems
* **Cloud Storage Buckets:** Document storage and large dataset repositories
* **External APIs:** Integration with other government systems and third-party data providers

#### 🧠 **Intelligent Query Processing**

The service implements sophisticated query processing capabilities:

```go
// Example of intelligent query routing

type QueryRouter struct {
    dwhEngine    QueryEngine     // For analytical queries
    opsDB        *sql.DB         // For operational data
    cacheLayer   cache.Cache     // For frequently accessed data
    legacySystems map[string]LegacyConnector
}

func (qr *QueryRouter) RouteQuery(ctx context.Context, req DataRequest) (*DataResponse, error) {
    // Analyze query requirements
    sources := qr.analyzeDataSources(req)
    
    // Route to appropriate data source(s)
    switch {
    case req.RequiresHistoricalData():
        return qr.queryDataWarehouse(ctx, req)
    case req.RequiresRealTimeData():
        return qr.queryOperationalDB(ctx, req)
    case req.RequiresMultiSourceJoin():
        return qr.executeDistributedQuery(ctx, req, sources)
    default:
        return qr.queryCachedData(ctx, req)
    }
}
```

#### 🎯 **Smart Data Orchestration**

**Query Optimization:**

* **Predicate Pushdown:** Filters are pushed down to the appropriate data sources to minimize data transfer
* **Join Optimization:** Intelligent decision-making about where to perform joins (at the source, in memory, or distributed)
* **Caching Strategy:** Multi-layer caching (Redis for hot data, local cache for frequently accessed queries)

**Data Transformation:**

* **Format Standardization:** Converts data from various source formats into consistent API responses
* **Data Enrichment:** Combines data from multiple sources to provide comprehensive responses
* **Real-time Processing:** Handles streaming data updates for time-sensitive procurement information

#### 🔄 **Advanced Data Processing Capabilities**

**Dynamic Query Generation:**

```sql
-- Example: Complex tender query with multiple filters

SELECT 
    t.tender_id,
    t.tender_name,
    t.budget_amount,
    k.klpd_name,
    p.package_details,
    d.document_urls

FROM dwh.tender_fact t

LEFT JOIN dwh.klpd_dim k ON t.klpd_id = k.klpd_id

LEFT JOIN ops.tender_packages p ON t.tender_id = p.tender_id

LEFT JOIN storage.documents d ON t.tender_id = d.entity_id

WHERE t.announcement_date >= $1 
  AND t.klpd_id = $2
  AND t.status IN ($3)
ORDER BY t.budget_amount DESC

LIMIT $4 OFFSET $5
```

**Cross-Database Joins:** The service can intelligently perform joins across different database systems:

* Fetch master data from the data warehouse
* Enrich with real-time status from operational databases
* Add document metadata from cloud storage systems

#### 🏛️ **Clean Architecture Implementation**

The Data Gateway Service follows clean architecture principles:

```
internal/
├── domain/              # Business logic and entities
│   ├── entities/        # Core business entities (Tender, RUP, Katalog)
│   ├── repositories/    # Data access interfaces
│   └── services/        # Business logic services
├── usecases/           # Application-specific business rules
│   ├── tender/         # Tender-related use cases
│   ├── rup/            # RUP-related use cases
│   └── katalog/        # Katalog-related use cases
├── interfaces/         # External interfaces
│   ├── http/           # HTTP handlers and middleware
│   ├── grpc/           # gRPC handlers (for internal communication)
│   └── database/       # Database implementations
└── infrastructure/     # External concerns
    ├── dwh/            # Data warehouse connectors
    ├── postgres/       # PostgreSQL implementations
    ├── cache/          # Caching implementations
    └── monitoring/     # Logging and metrics
```

#### 🔧 **Flexible Data Source Adapters**

The service uses a plugin-like architecture for data sources:

```go

type DataSourceAdapter interface {
    Connect(ctx context.Context, config Config) error
    Query(ctx context.Context, query Query) (*Result, error)
    HealthCheck(ctx context.Context) error
    Close() error
}

// Implementations for different data sources

type IcebergAdapter struct { /* ... */ }
type PostgreSQLAdapter struct { /* ... */ }
type LegacySQLServerAdapter struct { /* ... */ }
type CloudStorageAdapter struct { /* ... */ }
```

### Architecture Flow

```mermaid

graph TB
    A[Government Partners] -->|HTTPS + Auth Token| B[Inaproc API Gateway]
    B -->|Static API Key + mTLS| C[Data Gateway Service]
    C -->|Query Engine| D[Iceberg + Nessie DWH]
    C -->|SQL Queries| E[CloudSQL PostgreSQL]
    C -->|Data Joins| F[Additional Data Sources]
    
    B --> G[Fusio Backend]
    G --> H[Authentication System]
    
    style A fill:#e1f5fe
    style B fill:#1976d2,color:#fff
    style C fill:#388e3c,color:#fff
    style D fill:#ff9800,color:#fff
    style E fill:#9c27b0,color:#fff
```

### Detailed Request Flow

#### 1. **Partner Authentication Flow**

```mermaid

sequenceDiagram
    participant P as Government Partner
    participant AG as API Gateway
    participant AS as Auth Service
    participant DGS as Data Gateway Service
    participant DWH as Data Warehouse
    
    P->>AG: POST /auth/token (long_term_token)
    AG->>AS: Validate long term token
    AS->>AG: Return short term token
    AG->>P: short_term_token (expires 1h)
    
    P->>AG: GET /tender (Bearer short_term_token)
    AG->>AS: Validate short term token
    AS->>AG: Token valid + partner permissions
    AG->>DGS: Forward request (Static API Key + mTLS)
    DGS->>DWH: Execute optimized query
    DWH->>DGS: Return results
    DGS->>AG: Formatted response
    AG->>P: JSON response with pagination
```

#### 2. **Data Query Decision Tree**

The Data Gateway Service implements intelligent routing based on query characteristics:

```mermaid

graph TD
    A[Incoming Query Request] --> B{Query Analysis}
    B -->|Historical Data| C[Route to Data Warehouse]
    B -->|Real-time Data| D[Route to CloudSQL]
    B -->|Document Access| E[Route to Cloud Storage]
    B -->|Complex Joins| F[Multi-Source Orchestration]
    
    C --> G[Iceberg Table Scan]
    D --> H[PostgreSQL Query]
    E --> I[Cloud Storage API]
    F --> J[Distributed Query Execution]
    
    G --> K[Data Transformation]
    H --> K
    I --> K
    J --> K
    
    K --> L[Response Formatting]
    L --> M[Cache Update]
    M --> N[Return to API Gateway]
```

#### 3. **Performance Optimization Strategies**

**Caching Layers:**

* **L1 Cache (In-Memory):** Frequently accessed reference data (KLPD mappings, categories)
* **L2 Cache (Redis):** Query results with TTL based on data freshness requirements
* **L3 Cache (CDN):** Static content and documentation

**Query Optimization:**

```go

type QueryOptimizer struct {
    statisticsCollector *StatisticsCollector
    costBasedOptimizer  *CostBasedOptimizer
    cacheManager        *CacheManager
}

func (qo *QueryOptimizer) OptimizeQuery(query Query) OptimizedQuery {
    // Analyze query patterns
    stats := qo.statisticsCollector.Analyze(query)
    
    // Cost-based optimization
    plan := qo.costBasedOptimizer.CreateExecutionPlan(query, stats)
    
    // Check cache availability
    if cached := qo.cacheManager.GetCachedResult(query.Hash()); cached != nil {
        return OptimizedQuery{UseCached: true, CacheKey: query.Hash()}
    }
    
    return OptimizedQuery{
        ExecutionPlan:     plan,
        EstimatedCost:     plan.Cost(),
        RecommendedCache:  plan.ShouldCache(),
    }
}
```

#### 4. **Error Handling and Resilience**

**Circuit Breaker Pattern:**

```go

type CircuitBreaker struct {
    state       State // CLOSED, OPEN, HALF_OPEN
    failures    int
    lastFailure time.Time
    threshold   int
    timeout     time.Duration
}

func (cb *CircuitBreaker) Execute(operation func() (interface{}, error)) (interface{}, error) {
    if cb.state == OPEN {
        if time.Since(cb.lastFailure) > cb.timeout {
            cb.state = HALF_OPEN
        } else {
            return nil, errors.New("circuit breaker is open")
        }
    }
    
    result, err := operation()
    if err != nil {
        cb.recordFailure()
        return nil, err
    }
    
    cb.recordSuccess()
    return result, nil
}
```

**Fallback Strategies:**

* Primary: Iceberg Data Warehouse
* Secondary: Cached PostgreSQL aggregates
* Tertiary: Static reference data
* Final: Error response with retry guidance

#### 5. **Data Consistency and ACID Properties**

**Read Consistency:**

* **Strong Consistency:** For critical procurement data (tender statuses, awards)
* **Eventual Consistency:** For analytical data and historical records
* **Cache Consistency:** TTL-based invalidation with manual purge capabilities

**Transaction Management:**

```go

type TransactionManager struct {
    primaryDB   *sql.DB
    replicaDBs  []*sql.DB
    coordinator *TwoPhaseCommitCoordinator
}

func (tm *TransactionManager) ExecuteDistributedQuery(ctx context.Context, query DistributedQuery) error {
    tx, err := tm.coordinator.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Execute on primary
    if err := tm.executeOnPrimary(ctx, tx, query.PrimaryPart); err != nil {
        return err
    }
    
    // Execute on replicas
    for _, replica := range tm.replicaDBs {
        if err := tm.executeOnReplica(ctx, tx, replica, query.ReplicaParts); err != nil {
            return err
        }
    }
    
    return tx.Commit()
}
```


## 🔧 Components

### 1. Inaproc API Gateway (Fusio-based)

**Technology Stack:**

* **Framework:** Fusio (PHP-based)
* **Database:** MySQL/PostgreSQL
* **Features:** API Management, Authentication, Rate Limiting, Analytics

**Key Responsibilities:**

* Partner authentication and authorization
* API endpoint management and routing
* Request/response transformation
* Rate limiting and throttling
* API documentation and developer portal

**Fusio Configuration Example:**

```yaml
# routes.yaml
"/tender":
  version: 1
  status: 4
  methods:
    GET:
      active: true
      public: false
      action: "Tender_Collection"
      costs: 1

"/tender/:tender_id":
  version: 1
  status: 4
  methods:
    GET:
      active: true
      public: false
      action: "Tender_Entity"
      costs: 2
```

**Authentication Schema:**

```php
// Custom authentication action

class LongTermTokenAction extends ActionAbstract
{
    public function handle(RequestInterface $request, ParametersInterface $configuration, ContextInterface $context): ResponseInterface
    {
        $longTermToken = $request->get('long_term_token');
        $clientId = $request->get('client_id');
        
        // Validate long-term token
        $partner = $this->partnerService->validateLongTermToken($longTermToken, $clientId);
        
        if (!$partner) {
            throw new StatusCode\UnauthorizedException('Invalid long-term token');
        }
        
        // Generate short-term token
        $shortTermToken = $this->tokenService->generateShortTermToken($partner);
        
        return $this->response->build(200, [], [
            'short_term_token' => $shortTermToken,
            'expires_in' => 3600,
            'token_type' => 'Bearer'
        ]);
    }
}
```

### 2. Data Gateway Service

**Technology Stack:**

* **Language:** Go 1.21+
* **Framework:** Chi Router
* **Database:** CloudSQL PostgreSQL
* **Architecture:** Clean Architecture with Interfaces

**Enhanced Directory Structure:**

```
data-gateway-service/
├── cmd/
│   ├── server/              # Main application entry point
│   │   └── main.go
│   ├── migrate/             # Database migration tool
│   │   └── main.go
│   └── worker/              # Background job processor
│       └── main.go
├── internal/
│   ├── domain/              # Business logic layer
│   │   ├── entities/        # Core business entities
│   │   │   ├── tender.go
│   │   │   ├── rup.go
│   │   │   └── katalog.go
│   │   ├── repositories/    # Repository interfaces
│   │   │   └── interfaces.go
│   │   └── services/        # Domain services
│   │       ├── tender_service.go
│   │       ├── query_optimizer.go
│   │       └── cache_manager.go
│   ├── usecases/           # Application business rules
│   │   ├── tender/
│   │   │   ├── get_tender_list.go
│   │   │   ├── get_tender_detail.go
│   │   │   └── search_tender.go
│   │   ├── rup/
│   │   └── katalog/
│   ├── interfaces/         # Interface adapters
│   │   ├── http/           # HTTP layer
│   │   │   ├── handlers/
│   │   │   ├── middleware/
│   │   │   └── routes/
│   │   ├── grpc/           # gRPC layer (for internal services)
│   │   └── repositories/   # Repository implementations
│   │       ├── dwh/        # Data warehouse repositories
│   │       ├── postgres/   # PostgreSQL repositories
│   │       └── cache/      # Cache repositories
│   └── infrastructure/     # External concerns
│       ├── database/       # Database connections
│       ├── monitoring/     # Metrics and logging
│       ├── security/       # Authentication & authorization
│       └── config/         # Configuration management
├── pkg/                    # Shared packages
│   ├── logger/            # Structured logging
│   ├── metrics/           # Prometheus metrics
│   ├── tracing/           # Distributed tracing
│   └── errors/            # Error handling utilities
├── scripts/               # Build and deployment scripts
├── migrations/            # Database migrations
├── tests/                 # Test suites
│   ├── unit/
│   ├── integration/
│   └── e2e/
└── configs/               # Configuration files
    ├── config.yaml
    ├── config.production.yaml
    └── config.development.yaml
```

**Core Interfaces and Implementation:**

```go
// Domain entities

type Tender struct {
    ID               string    `json:"tender_id" db:"tender_id"`
    KLPDID           int       `json:"klpd_id" db:"klpd_id"`
    KLPDName         string    `json:"klpd_name" db:"klpd_name"`
    Name             string    `json:"tender_name" db:"tender_name"`
    Description      string    `json:"description" db:"description"`
    AnnouncementDate time.Time `json:"announcement_date" db:"announcement_date"`
    ClosingDate      time.Time `json:"closing_date" db:"closing_date"`
    BudgetAmount     float64   `json:"budget_amount" db:"budget_amount"`
    Method           string    `json:"procurement_method" db:"procurement_method"`
    Status           string    `json:"status" db:"status"`
    Documents        []Document `json:"documents,omitempty"`
    Timeline         []TimelineEvent `json:"timeline,omitempty"`
}

// Repository interfaces

type TenderRepository interface {
    GetTenders(ctx context.Context, params TenderQueryParams) (*TenderCollection, error)
    GetTenderByID(ctx context.Context, id string) (*Tender, error)
    SearchTenders(ctx context.Context, query string, filters TenderFilters) (*TenderCollection, error)
    GetTenderStatistics(ctx context.Context, filters TenderFilters) (*TenderStats, error)
}

type QueryEngine interface {
    ExecuteQuery(ctx context.Context, query string, params map[string]interface{}) (*QueryResult, error)
    ExecuteComplexQuery(ctx context.Context, query ComplexQuery) (*QueryResult, error)
    BuildOptimizedQuery(entity string, filters map[string]interface{}) (*OptimizedQuery, error)
    EstimateQueryCost(query string) (*QueryCost, error)
}

// Multi-source data aggregator

type DataAggregator interface {
    AggregateFromMultipleSources(ctx context.Context, request AggregationRequest) (*AggregatedResponse, error)
    JoinAcrossDataSources(ctx context.Context, joinSpec JoinSpecification) (*JoinResult, error)
    ExecuteDistributedQuery(ctx context.Context, distributedQuery DistributedQuery) (*DistributedResult, error)
}
```

**Advanced Query Processing:**

```go

type QueryProcessor struct {
    dwhConnector     DWHConnector
    postgresDB       *sql.DB
    cacheLayer       Cache
    queryOptimizer   *QueryOptimizer
    circuitBreaker   *CircuitBreaker
    metricsCollector *MetricsCollector
}

func (qp *QueryProcessor) ProcessTenderQuery(ctx context.Context, params TenderQueryParams) (*TenderResponse, error) {
    // Start tracing
    span, ctx := opentracing.StartSpanFromContext(ctx, "process_tender_query")
    defer span.Finish()
    
    // Validate and sanitize parameters
    if err := qp.validateParams(params); err != nil {
        return nil, fmt.Errorf("parameter validation failed: %w", err)
    }
    
    // Check cache first
    cacheKey := qp.generateCacheKey("tender", params)
    if cached, err := qp.cacheLayer.Get(ctx, cacheKey); err == nil {
        qp.metricsCollector.RecordCacheHit("tender")
        return qp.deserializeTenderResponse(cached), nil
    }
    
    // Determine optimal data source
    strategy := qp.queryOptimizer.DetermineQueryStrategy(params)
    
    var result *TenderResponse
    var err error
    
    switch strategy.PreferredSource {
    case DataWarehouse:
        result, err = qp.queryDataWarehouse(ctx, params, strategy)
    case OperationalDB:
        result, err = qp.queryPostgreSQL(ctx, params, strategy)
    case HybridQuery:
        result, err = qp.executeHybridQuery(ctx, params, strategy)
    default:
        return nil, fmt.Errorf("unsupported query strategy: %v", strategy.PreferredSource)
    }
    
    if err != nil {
        qp.metricsCollector.RecordQueryError("tender", err)
        return nil, err
    }
    
    // Cache successful results
    if strategy.ShouldCache {
        qp.cacheLayer.Set(ctx, cacheKey, qp.serializeTenderResponse(result), strategy.CacheTTL)
    }
    
    qp.metricsCollector.RecordQuerySuccess("tender", strategy.PreferredSource)
    return result, nil
}
```

**Service Configuration:**

```yaml
# config.yaml

server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s

database:
  postgres:
    host: "localhost"
    port: 5432
    database: "inaproc_gateway"
    username: "gateway_user"
    password: "${POSTGRES_PASSWORD}"
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: 5m

data_warehouse:
  type: "trino"
  connection_string: "jdbc:trino://trino-coordinator:8080/iceberg"
  catalog: "inaproc_catalog"
  schema: "procurement"
  query_timeout: 300s

cache:
  redis:
    addresses: ["redis:6379"]
    password: "${REDIS_PASSWORD}"
    db: 0
    pool_size: 10
    min_idle_conns: 2

security:
  api_key: "${DATA_GATEWAY_API_KEY}"
  tls:
    cert_file: "/etc/certs/server.crt"
    key_file: "/etc/certs/server.key"
    ca_file: "/etc/certs/ca.crt"

monitoring:
  enable_metrics: true
  metrics_port: 9090
  enable_tracing: true
  jaeger_endpoint: "http://jaeger:14268/api/traces"
  log_level: "info"

rate_limiting:
  requests_per_minute: 1000
  burst_size: 100
  cleanup_interval: 60s
```

### 3. Data Warehouse (Iceberg + Nessie)

**Technology Stack:**

* **Table Format:** Apache Iceberg
* **Catalog:** Project Nessie
* **Query Engine:** Trino/Presto
* **Storage:** Cloud Storage (GCS/S3)

**Iceberg Table Schema Examples:**

```sql
-- Tender fact table

CREATE TABLE iceberg.procurement.tender_fact (
    tender_id VARCHAR NOT NULL,
    klpd_id INTEGER NOT NULL,
    tender_name VARCHAR(500) NOT NULL,
    description TEXT,
    announcement_date DATE NOT NULL,
    closing_date DATE,
    budget_amount DECIMAL(20,2),
    procurement_method VARCHAR(100),
    status VARCHAR(50),
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    partition_year INTEGER,
    partition_month INTEGER
) 
PARTITIONED BY (partition_year, partition_month)
STORED AS PARQUET;

-- KLPD dimension table

CREATE TABLE iceberg.procurement.klpd_dim (
    klpd_id INTEGER NOT NULL,
    klpd_code VARCHAR(20),
    klpd_name VARCHAR(200) NOT NULL,
    parent_klpd_id INTEGER,
    level INTEGER,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
)
STORED AS PARQUET;
```

**Nessie Branch Management:**

```yaml
# Nessie configuration for data versioning

nessie:
  branches:
    - name: "main"
      description: "Production data branch"
      protection_rules:
        - require_review: true
        - restrict_push: ["production"]
    - name: "staging"
      description: "Staging environment data"
      merge_strategy: "fast_forward"
    - name: "development"
      description: "Development and testing"
      auto_cleanup: true
      retention_days: 30

  tags:
    - name: "v2024.12"
      branch: "main"
      description: "December 2024 data snapshot"
    - name: "monthly_backup"
      branch: "main"
      auto_create: true
      schedule: "0 0 1 * *"  # First day of each month
```

**Query Engine Optimization:**

```sql
-- Example of optimized query with predicate pushdown

SELECT 
    t.tender_id,
    t.tender_name,
    t.budget_amount,
    k.klpd_name,
    CASE 
        WHEN t.budget_amount > 50000000000 THEN 'Large Scale'
        WHEN t.budget_amount > 10000000000 THEN 'Medium Scale'
        ELSE 'Small Scale'
    END as project_scale

FROM iceberg.procurement.tender_fact t

INNER JOIN iceberg.procurement.klpd_dim k 
    ON t.klpd_id = k.klpd_id

WHERE t.partition_year = 2024
    AND t.partition_month = 12
    AND t.klpd_id = 66
    AND t.status IN ('ACTIVE', 'EVALUATION')
ORDER BY t.budget_amount DESC

LIMIT 50;
```


## 📊 Data Catalog

### Available Datasets

#### 1. Tender Data

| Field | Type | Description | Example |
|----|----|----|----|
| `tender_id` | STRING | Unique tender identifier | "TND-2024-001234" |
| `klpd_id` | INTEGER | Government entity ID | 66 (Kementrian PU) |
| `tender_name` | STRING | Tender project name | "Pembangunan Jalan Tol" |
| `announcement_date` | DATE | Tender announcement date | "2024-01-15" |
| `closing_date` | DATE | Bid submission deadline | "2024-02-15" |
| `budget_amount` | DECIMAL | Estimated budget | 15000000000.00 |
| `procurement_method` | STRING | Procurement methodology | "Tender Terbuka" |
| `status` | STRING | Current tender status | "ACTIVE", "CLOSED", "AWARDED" |

#### 2. RUP (Rencana Umum Pengadaan) Data

| Field | Type | Description | Example |
|----|----|----|----|
| `rup_id` | STRING | RUP unique identifier | "RUP-2024-567890" |
| `klpd_id` | INTEGER | Government entity ID | 66 |
| `package_name` | STRING | Procurement package name | "Pengadaan Laptop" |
| `planned_date` | DATE | Planned procurement date | "2024-06-01" |
| `estimated_value` | DECIMAL | Estimated procurement value | 500000000.00 |
| `procurement_type` | STRING | Type of procurement | "Barang", "Jasa", "Konstruksi" |
| `funding_source` | STRING | Budget source | "APBN", "APBD" |

#### 3. Katalog V6 Data

| Field | Type | Description | Example |
|----|----|----|----|
| `catalog_id` | STRING | Catalog item identifier | "CAT-V6-789123" |
| `product_name` | STRING | Product/service name | "Laptop Dell Latitude 5520" |
| `category` | STRING | Product category | "IT Equipment" |
| `specification` | JSON | Detailed specifications | {"cpu": "Intel i7", "ram": "16GB"} |
| `price` | DECIMAL | Catalog price | 15000000.00 |
| `vendor_id` | STRING | Vendor identifier | "VND-001234" |
| `validity_period` | DATE | Price validity end date | "2024-12-31" |


## 🔌 API Specifications

### Base URL

```
Production: https://api.inaproc.lkpp.go.id/v1

Staging: https://staging-api.inaproc.lkpp.go.id/v1
```

### Authentication Headers

```http

Authorization: Bearer {short_term_token}
X-API-Client-ID: {partner_client_id}
```

### 1. Tender Endpoints

#### Get Tender List

```http

GET /tender
```

**Query Parameters:**

| Parameter | Type | Required | Description | Example |
|----|----|----|----|----|
| `month` | integer | No | Filter by month (1-12) | 12 |
| `year` | integer | No | Filter by year | 2024 |
| `klpd_id` | integer | No | Government entity ID | 66 |
| `status` | string | No | Tender status | "ACTIVE" |
| `min_budget` | number | No | Minimum budget filter | 1000000000 |
| `max_budget` | number | No | Maximum budget filter | 10000000000 |
| `page` | integer | No | Page number (default: 1) | 1 |
| `limit` | integer | No | Items per page (default: 20, max: 100) | 20 |

**Response Example:**

```json
{
  "status": "success",
  "data": {
    "items": [
      {
        "tender_id": "TND-2024-001234",
        "klpd_id": 66,
        "klpd_name": "Kementerian Pekerjaan Umum dan Perumahan Rakyat",
        "tender_name": "Pembangunan Jalan Tol Jakarta-Surabaya Seksi 3",
        "announcement_date": "2024-01-15",
        "closing_date": "2024-02-15",
        "budget_amount": 15000000000.00,
        "procurement_method": "Tender Terbuka",
        "status": "ACTIVE"
      }
    ],
    "pagination": {
      "current_page": 1,
      "total_pages": 45,
      "total_items": 892,
      "per_page": 20
    }
  },
  "meta": {
    "timestamp": "2024-12-19T10:30:00Z",
    "request_id": "req_abc123def456"
  }
}
```

#### Get Tender Details

```http

GET /tender/{tender_id}
```

**Response Example:**

```json
{
  "status": "success",
  "data": {
    "tender_id": "TND-2024-001234",
    "klpd_id": 66,
    "klpd_name": "Kementerian Pekerjaan Umum dan Perumahan Rakyat",
    "tender_name": "Pembangunan Jalan Tol Jakarta-Surabaya Seksi 3",
    "description": "Pembangunan infrastruktur jalan tol dengan panjang 45 km",
    "announcement_date": "2024-01-15",
    "closing_date": "2024-02-15",
    "budget_amount": 15000000000.00,
    "procurement_method": "Tender Terbuka",
    "status": "ACTIVE",
    "documents": [
      {
        "document_type": "RKS",
        "document_name": "Rencana Kerja dan Syarat-syarat.pdf",
        "download_url": "/documents/tender/TND-2024-001234/rks.pdf"
      }
    ],
    "timeline": [
      {
        "event": "Pengumuman",
        "date": "2024-01-15",
        "status": "completed"
      },
      {
        "event": "Penutupan Pendaftaran",
        "date": "2024-02-15",
        "status": "upcoming"
      }
    ]
  }
}
```

### 2. RUP Endpoints

#### Get RUP List

```http

GET /rup
```

**Query Parameters:**

| Parameter | Type | Required | Description |
|----|----|----|----|
| `month` | integer | No | Filter by planned month |
| `year` | integer | No | Filter by planned year |
| `klpd_id` | integer | No | Government entity ID |
| `procurement_type` | string | No | Type: "Barang", "Jasa", "Konstruksi" |
| `min_value` | number | No | Minimum estimated value |
| `max_value` | number | No | Maximum estimated value |

### 3. Katalog V6 Endpoints

#### Get Katalog Items

```http

GET /katalog
```

**Query Parameters:**

| Parameter | Type | Required | Description |
|----|----|----|----|
| `category` | string | No | Product category |
| `vendor_id` | string | No | Vendor identifier |
| `min_price` | number | No | Minimum price filter |
| `max_price` | number | No | Maximum price filter |
| `search` | string | No | Search in product name |

### Error Responses

```json
{
  "status": "error",
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "Invalid klpd_id parameter",
    "details": {
      "parameter": "klpd_id",
      "value": "invalid_value",
      "expected": "integer"
    }
  },
  "meta": {
    "timestamp": "2024-12-19T10:30:00Z",
    "request_id": "req_abc123def456"
  }
}
```

**HTTP Status Codes:**

* `200` - Success
* `400` - Bad Request (Invalid parameters)
* `401` - Unauthorized (Invalid token)
* `403` - Forbidden (Insufficient permissions)
* `404` - Not Found
* `429` - Too Many Requests (Rate limit exceeded)
* `500` - Internal Server Error


## 🔐 Authentication & Security

### Dual Authentication Flow

#### 1. Long-term Token Exchange

Government partners receive long-term tokens via secure email channels.

```http

POST /auth/token

Content-Type: application/json

{
  "long_term_token": "ltt_abc123def456...",
  "client_id": "partner_ministry_pu"
}
```

**Response:**

```json
{
  "short_term_token": "stt_xyz789abc123...",
  "expires_in": 3600,
  "token_type": "Bearer"
}
```

#### 2. API Request with Short-term Token

```http

GET /tender?klpd_id=66&month=12&year=2024

Authorization: Bearer stt_xyz789abc123...
X-API-Client-ID: partner_ministry_pu
```

### Security Measures

#### API Gateway ↔ Data Gateway Service

* **Static API Key:** Shared secret for service-to-service communication
* **mTLS (Mutual TLS):** Certificate-based authentication at infrastructure level
* **Request Validation:** Schema validation and sanitization

#### Data Protection

* **Encryption at Rest:** All sensitive data encrypted in databases
* **Encryption in Transit:** HTTPS/TLS for all communications
* **Access Logging:** Comprehensive audit trail for all API requests
* **Rate Limiting:** Configurable limits per partner and endpoint


## 🚀 Deployment

### Infrastructure Requirements

#### API Gateway (Fusio)

```yaml

Resources:
  - CPU: 2 vCPU
  - Memory: 4 GB RAM
  - Storage: 50 GB SSD
  - Database: CloudSQL MySQL 8.0
```

#### Data Gateway Service

```yaml

Resources:
  - CPU: 4 vCPU
  - Memory: 8 GB RAM
  - Storage: 100 GB SSD
  - Database: CloudSQL PostgreSQL 14
```

### Docker Deployment

#### API Gateway

```dockerfile

FROM fusio/fusio:latest

COPY ./config /var/www/html/
ENV FUSIO_CONNECTION="pdo-mysql://user:pass@mysql:3306/fusio"
EXPOSE 80
```

#### Data Gateway Service

```dockerfile

FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY . .
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o data-gateway ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/data-gateway .
EXPOSE 8080

CMD ["./data-gateway"]
```

### Docker Compose

```yaml

version: '3.8'
services:
  api-gateway:
    image: inaproc/api-gateway:latest
    ports:
      - "80:80"
    environment:
      - FUSIO_CONNECTION=pdo-mysql://fusio:password@mysql:3306/fusio
    depends_on:
      - mysql
      - data-gateway-service

  data-gateway-service:
    image: inaproc/data-gateway-service:latest
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://user:password@postgres:5432/datagateway
      - DWH_CONNECTION_STRING=jdbc:trino://trino:8080/iceberg
    depends_on:
      - postgres

  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: fusio
      MYSQL_USER: fusio
      MYSQL_PASSWORD: password

  postgres:
    image: postgres:14
    environment:
      POSTGRES_DB: datagateway
      POSTGRES_USER: user
      POSTGRES_PASSWORD: password
```


## 👨‍💻 Development Guide

### Setting Up Development Environment

#### Prerequisites

* Go 1.21+
* PHP 8.2+
* Docker & Docker Compose
* MySQL 8.0
* PostgreSQL 14

#### API Gateway Setup (Fusio)

```bash
# Clone and setup Fusio

git clone https://github.com/your-org/inaproc-api-gateway.git

cd inaproc-api-gateway

# Install dependencies

composer install

# Setup environment

cp .env.example .env

# Run migrations

php bin/fusio migrate

# Create admin user

php bin/fusio adduser

# Start development server

php -S 127.0.0.1:8080 -t public
```

#### Data Gateway Service Setup

```bash
# Clone repository

git clone https://github.com/your-org/data-gateway-service.git

cd data-gateway-service

# Install dependencies

go mod download

# Setup environment

cp configs/config.example.yaml configs/config.yaml

# Run database migrations

go run cmd/migrate/main.go

# Start development server

go run cmd/server/main.go
```

### Code Quality Standards

#### Go Code Standards

```go
// Example service implementation

type TenderService struct {
    repo       interfaces.TenderRepository
    logger     *zap.Logger
    validator  *validator.Validate
}

func (s *TenderService) GetTenders(ctx context.Context, params TenderParams) (*TenderResponse, error) {
    // Validate input parameters
    if err := s.validator.Struct(params); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }

    // Execute repository method
    tenders, err := s.repo.GetTenders(ctx, params)
    if err != nil {
        s.logger.Error("failed to get tenders", zap.Error(err))
        return nil, fmt.Errorf("failed to retrieve tenders: %w", err)
    }

    return &TenderResponse{
        Items: tenders,
        Pagination: s.buildPagination(params, len(tenders)),
    }, nil
}
```

#### Testing Strategy

```bash
# Run unit tests

go test ./...

# Run integration tests

go test -tags=integration ./...

# Run with coverage

go test -cover ./...

# Load testing

k6 run tests/load/api_load_test.js
```


## 📈 Monitoring & Analytics

### Metrics Collection

* **Request/Response Metrics:** Latency, throughput, error rates
* **Business Metrics:** API usage by partner, popular endpoints
* **Infrastructure Metrics:** CPU, memory, database performance

### Logging Strategy

```go
// Structured logging example

logger.Info("tender request processed",
    zap.String("request_id", requestID),
    zap.String("partner_id", partnerID),
    zap.String("endpoint", "/tender"),
    zap.Duration("duration", duration),
    zap.Int("result_count", len(results)),
)
```

### Health Checks

```http

GET /health

GET /ready

GET /metrics
```


### Development Workflow



1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request


