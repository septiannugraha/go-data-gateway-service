package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"go-data-gateway/internal/clients"
	"go-data-gateway/internal/response"
)

// RUPHandler handles RUP (Rencana Umum Pengadaan) queries from BigQuery
type RUPHandler struct {
	bigquery *clients.BigQueryClient
	logger   *zap.Logger
}

// NewRUPHandler creates a new RUP handler
func NewRUPHandler(bigquery *clients.BigQueryClient, logger *zap.Logger) *RUPHandler {
	return &RUPHandler{
		bigquery: bigquery,
		logger:   logger,
	}
}

// RUPResponse represents the response structure for RUP data
type RUPResponse struct {
	KdRUP        string  `json:"kd_rup"`
	NamaPaket    string  `json:"nama_paket"`
	Pagu         float64 `json:"pagu"`
	Tahun        string  `json:"tahun"`
	KdSatker     string  `json:"kd_satker"`
	NamaSatker   string  `json:"nama_satker"`
	KdKlpd       string  `json:"kd_klpd"`
	NamaKlpd     string  `json:"nama_klpd"`
	JenisPengadaan string `json:"jenis_pengadaan"`
	MetodePengadaan string `json:"metode_pengadaan"`
	SumberDana   string  `json:"sumber_dana"`
	TahunAnggaran string `json:"tahun_anggaran"`
	CreatedDate  string  `json:"created_date"`
}

// List handles GET /api/v1/rup
func (h *RUPHandler) List(w http.ResponseWriter, r *http.Request) {
	if h.bigquery == nil {
		response.Error(w, "BigQuery service not available", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	params := r.URL.Query()
	limit := 100
	offset := 0

	if l := params.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}

	if o := params.Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Build query - adjust table name based on your actual BigQuery dataset
	query := fmt.Sprintf(`
		SELECT
			kd_rup,
			nama_paket,
			pagu,
			tahun_anggaran,
			kd_satker,
			nama_satker,
			kd_klpd,
			nama_klpd,
			jenis_pengadaan,
			metode_pengadaan,
			sumber_dana,
			FORMAT_TIMESTAMP('%%Y-%%m-%%dT%%H:%%M:%%S', created_date) as created_date
		FROM %s.rup_data
		ORDER BY created_date DESC
		LIMIT %d OFFSET %d
	`, "`lkpp-dataset`", limit, offset) // Replace with actual dataset name

	results, err := h.bigquery.Query(r.Context(), query)
	if err != nil {
		h.logger.Error("Failed to query RUP data", zap.Error(err))
		response.ErrorWithDetails(w, "Failed to fetch RUP data", err.Error(), http.StatusInternalServerError)
		return
	}

	// Also get total count for pagination
	countQuery := fmt.Sprintf("SELECT COUNT(*) as total FROM `%s.rup_data`", "lkpp-dataset")
	countResult, err := h.bigquery.Query(r.Context(), countQuery)
	if err != nil {
		h.logger.Warn("Failed to get total count", zap.Error(err))
	}

	var total int64 = int64(len(results))
	if len(countResult) > 0 {
		if v, ok := countResult[0]["total"].(int64); ok {
			total = v
		}
	}

	// Calculate pagination
	page := (offset / limit) + 1

	response.Success(w, results, &response.Meta{
		Page:    page,
		PerPage: limit,
		Total:   int(total),
	})
}

// GetByID handles GET /api/v1/rup/:id
func (h *RUPHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	if h.bigquery == nil {
		response.Error(w, "BigQuery service not available", http.StatusServiceUnavailable)
		return
	}

	// Get ID from URL path
	idPath := strings.TrimPrefix(r.URL.Path, "/api/v1/rup/")
	if idPath == "" {
		response.Error(w, "RUP ID is required", http.StatusBadRequest)
		return
	}

	// Sanitize ID to prevent SQL injection
	id := strings.ReplaceAll(idPath, "'", "''")

	query := fmt.Sprintf(`
		SELECT
			kd_rup,
			nama_paket,
			pagu,
			tahun_anggaran,
			kd_satker,
			nama_satker,
			kd_klpd,
			nama_klpd,
			jenis_pengadaan,
			metode_pengadaan,
			sumber_dana,
			lokasi,
			awal_pengadaan,
			akhir_pengadaan,
			awal_pelaksanaan,
			akhir_pelaksanaan,
			spesifikasi,
			FORMAT_TIMESTAMP('%%Y-%%m-%%dT%%H:%%M:%%S', created_date) as created_date,
			FORMAT_TIMESTAMP('%%Y-%%m-%%dT%%H:%%M:%%S', updated_date) as updated_date
		FROM %s.rup_data
		WHERE kd_rup = '%s'
		LIMIT 1
	`, "`lkpp-dataset`", id) // Replace with actual dataset name

	results, err := h.bigquery.Query(r.Context(), query)
	if err != nil {
		h.logger.Error("Failed to query RUP by ID",
			zap.String("id", id),
			zap.Error(err))
		response.ErrorWithDetails(w, "Failed to fetch RUP data", err.Error(), http.StatusInternalServerError)
		return
	}

	if len(results) == 0 {
		response.Error(w, "RUP not found", http.StatusNotFound)
		return
	}

	response.Success(w, results[0], nil)
}

// Search handles POST /api/v1/rup/search
func (h *RUPHandler) Search(w http.ResponseWriter, r *http.Request) {
	if h.bigquery == nil {
		response.Error(w, "BigQuery service not available", http.StatusServiceUnavailable)
		return
	}

	var req struct {
		Keyword  string `json:"keyword"`
		Tahun    string `json:"tahun"`
		KdSatker string `json:"kd_satker"`
		MinPagu  float64 `json:"min_pagu"`
		MaxPagu  float64 `json:"max_pagu"`
		Limit    int    `json:"limit"`
		Offset   int    `json:"offset"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Default values
	if req.Limit == 0 || req.Limit > 1000 {
		req.Limit = 100
	}

	// Build WHERE clauses
	var conditions []string

	if req.Keyword != "" {
		keyword := strings.ReplaceAll(req.Keyword, "'", "''")
		conditions = append(conditions, fmt.Sprintf(
			"(LOWER(nama_paket) LIKE LOWER('%%%s%%') OR LOWER(spesifikasi) LIKE LOWER('%%%s%%'))",
			keyword, keyword,
		))
	}

	if req.Tahun != "" {
		conditions = append(conditions, fmt.Sprintf("tahun_anggaran = '%s'",
			strings.ReplaceAll(req.Tahun, "'", "''")))
	}

	if req.KdSatker != "" {
		conditions = append(conditions, fmt.Sprintf("kd_satker = '%s'",
			strings.ReplaceAll(req.KdSatker, "'", "''")))
	}

	if req.MinPagu > 0 {
		conditions = append(conditions, fmt.Sprintf("pagu >= %f", req.MinPagu))
	}

	if req.MaxPagu > 0 {
		conditions = append(conditions, fmt.Sprintf("pagu <= %f", req.MaxPagu))
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT
			kd_rup,
			nama_paket,
			pagu,
			tahun_anggaran,
			kd_satker,
			nama_satker,
			kd_klpd,
			nama_klpd,
			jenis_pengadaan,
			metode_pengadaan,
			sumber_dana,
			FORMAT_TIMESTAMP('%%Y-%%m-%%dT%%H:%%M:%%S', created_date) as created_date
		FROM %s.rup_data
		%s
		ORDER BY created_date DESC
		LIMIT %d OFFSET %d
	`, "`lkpp-dataset`", whereClause, req.Limit, req.Offset)

	results, err := h.bigquery.Query(r.Context(), query)
	if err != nil {
		h.logger.Error("Failed to search RUP data",
			zap.String("query", query),
			zap.Error(err))
		response.ErrorWithDetails(w, "Failed to search RUP data", err.Error(), http.StatusInternalServerError)
		return
	}

	// Get total count for pagination
	countQuery := fmt.Sprintf(
		"SELECT COUNT(*) as total FROM `lkpp-dataset`.rup_data %s",
		whereClause,
	)

	countResult, _ := h.bigquery.Query(r.Context(), countQuery)
	var total int64 = int64(len(results))
	if len(countResult) > 0 {
		if v, ok := countResult[0]["total"].(int64); ok {
			total = v
		}
	}

	// Create meta with additional info in data itself
	meta := &response.Meta{
		Total:   int(total),
		Page:    (req.Offset / req.Limit) + 1,
		PerPage: req.Limit,
	}

	// Wrap results with filter info
	responseData := map[string]interface{}{
		"results":  results,
		"filtered": len(conditions) > 0,
		"filters_applied": map[string]interface{}{
			"keyword":   req.Keyword,
			"tahun":     req.Tahun,
			"kd_satker": req.KdSatker,
			"min_pagu":  req.MinPagu,
			"max_pagu":  req.MaxPagu,
		},
	}

	response.Success(w, responseData, meta)
}