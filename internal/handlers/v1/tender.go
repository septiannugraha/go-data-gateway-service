package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"go-data-gateway/internal/datasource"
	"go-data-gateway/internal/response"
)

// TenderHandler handles tender-related endpoints
type TenderHandler struct {
	dataSource datasource.DataSource
	logger     *zap.Logger
}

// NewTenderHandler creates a new tender handler
func NewTenderHandler(dataSource datasource.DataSource, logger *zap.Logger) *TenderHandler {
	return &TenderHandler{
		dataSource: dataSource,
		logger:     logger,
	}
}

// List handles GET /api/v1/tender
func (h *TenderHandler) List(w http.ResponseWriter, r *http.Request) {
	if h.dataSource == nil {
		response.Error(w, "Data source not configured", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	status := r.URL.Query().Get("status")
	sortBy := r.URL.Query().Get("sort_by")
	if sortBy == "" {
		sortBy = "tanggal_buat_paket"
	}

	order := r.URL.Query().Get("order")
	if order == "" {
		order = "DESC"
	}

	// Build SQL query
	query := `
		SELECT
			tender_id,
			nama_paket,
			nilai_pagu,
			metode_pengadaan,
			tahun_anggaran,
			status_tender,
			tanggal_buat_paket,
			tanggal_pengumuman,
			provinsi,
			jenis_pengadaan,
			nama_kl,
			nilai_kontrak,
			satuan_kerja
		FROM nessie_iceberg.tender_data
		WHERE 1=1
	`

	// Add status filter if provided
	if status != "" {
		query += fmt.Sprintf(" AND status_tender = '%s'", status)
	}

	// Add sorting and pagination
	query += fmt.Sprintf(" ORDER BY %s %s LIMIT %d OFFSET %d", sortBy, order, limit, offset)

	// Execute query
	opts := &datasource.QueryOptions{
		Limit:  limit,
		Offset: offset,
	}

	result, err := h.dataSource.ExecuteQuery(r.Context(), query, opts)
	if err != nil {
		h.logger.Error("Failed to fetch tenders", zap.Error(err))
		response.Error(w, "Failed to fetch tender data", http.StatusInternalServerError)
		return
	}

	// Add pagination meta
	meta := &response.Meta{
		Page:    (offset / limit) + 1,
		PerPage: limit,
		Total:   result.Count,
	}

	response.Success(w, result.Data, meta)
}

// GetByID handles GET /api/v1/tender/{id}
func (h *TenderHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	if h.dataSource == nil {
		response.Error(w, "Data source not configured", http.StatusServiceUnavailable)
		return
	}

	tenderID := chi.URLParam(r, "id")
	if tenderID == "" {
		response.Error(w, "Tender ID is required", http.StatusBadRequest)
		return
	}

	query := fmt.Sprintf(`
		SELECT * FROM nessie_iceberg.tender_data
		WHERE tender_id = '%s'
		LIMIT 1
	`, tenderID)

	result, err := h.dataSource.ExecuteQuery(r.Context(), query, nil)
	if err != nil {
		h.logger.Error("Failed to fetch tender", zap.Error(err))
		response.Error(w, "Failed to fetch tender data", http.StatusInternalServerError)
		return
	}

	if len(result.Data) == 0 {
		response.Error(w, "Tender not found", http.StatusNotFound)
		return
	}

	response.Success(w, result.Data[0], nil)
}

// Search handles POST /api/v1/tender/search
func (h *TenderHandler) Search(w http.ResponseWriter, r *http.Request) {
	if h.dataSource == nil {
		response.Error(w, "Data source not configured", http.StatusServiceUnavailable)
		return
	}

	// Parse search criteria from request body
	var searchCriteria map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&searchCriteria); err != nil {
		response.Error(w, "Invalid search criteria", http.StatusBadRequest)
		return
	}

	// Build query based on search criteria
	query := `SELECT * FROM nessie_iceberg.tender_data WHERE 1=1`

	// Add filters dynamically
	for field, value := range searchCriteria {
		if field == "limit" || field == "offset" {
			continue
		}
		query += fmt.Sprintf(" AND %s = '%v'", field, value)
	}

	// Add default limit
	query += " LIMIT 100"

	result, err := h.dataSource.ExecuteQuery(r.Context(), query, nil)
	if err != nil {
		h.logger.Error("Search failed", zap.Error(err))
		response.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	response.Success(w, result, nil)
}
