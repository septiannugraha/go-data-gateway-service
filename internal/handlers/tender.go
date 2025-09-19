package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"go-data-gateway/internal/clients"
)

// TenderHandler handles tender-related endpoints (Dremio/Iceberg)
type TenderHandler struct {
	dremio *clients.DremioClient
	logger *zap.Logger
}

// NewTenderHandler creates a new tender handler
func NewTenderHandler(dremio *clients.DremioClient, logger *zap.Logger) *TenderHandler {
	return &TenderHandler{
		dremio: dremio,
		logger: logger,
	}
}

// List returns a list of tenders with pagination
func (h *TenderHandler) List(c *gin.Context) {
	if h.dremio == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Dremio service not configured",
		})
		return
	}

	// Parse query parameters
	limit := c.DefaultQuery("limit", "100")
	offset := c.DefaultQuery("offset", "0")
	status := c.Query("status")
	sortBy := c.DefaultQuery("sort_by", "tanggal_buat_paket")
	order := c.DefaultQuery("order", "DESC")

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
			jenis_pengadaan
		FROM nessie_iceberg.tender_data
		WHERE 1=1
	`

	// Add status filter if provided
	if status != "" {
		query += fmt.Sprintf(" AND status_tender = '%s'", status)
	}

	// Add sorting
	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, order)

	// Add pagination
	query += fmt.Sprintf(" LIMIT %s OFFSET %s", limit, offset)

	// Execute query
	results, err := h.dremio.Query(c.Request.Context(), query)
	if err != nil {
		h.logger.Error("Failed to fetch tenders", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch tender data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   results,
		"count":  len(results),
		"limit":  limit,
		"offset": offset,
	})
}

// GetByID returns a single tender by ID
func (h *TenderHandler) GetByID(c *gin.Context) {
	if h.dremio == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Dremio service not configured",
		})
		return
	}

	tenderID := c.Param("id")

	query := fmt.Sprintf(`
		SELECT
			tender_id,
			nama_paket,
			nilai_pagu,
			nilai_hps,
			metode_pengadaan,
			tahun_anggaran,
			status_tender,
			tanggal_pembuatan,
			tanggal_pengumuman,
			tanggal_penutupan,
			lokasi,
			kategori,
			satuan_kerja,
			kualifikasi_usaha,
			syarat_kualifikasi,
			peserta_tender
		FROM nessie_iceberg.tender_data
		WHERE tender_id = '%s'
		LIMIT 1
	`, tenderID)

	results, err := h.dremio.Query(c.Request.Context(), query)
	if err != nil {
		h.logger.Error("Failed to fetch tender", zap.Error(err), zap.String("tender_id", tenderID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch tender data",
		})
		return
	}

	if len(results) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Tender not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": results[0],
	})
}

// Search performs advanced search on tenders
func (h *TenderHandler) Search(c *gin.Context) {
	if h.dremio == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Dremio service not configured",
		})
		return
	}

	var request struct {
		Keyword      string   `json:"keyword"`
		MinValue     float64  `json:"min_value"`
		MaxValue     float64  `json:"max_value"`
		Status       []string `json:"status"`
		Kategori     []string `json:"kategori"`
		TahunAnggaran int     `json:"tahun_anggaran"`
		Lokasi       string   `json:"lokasi"`
		StartDate    string   `json:"start_date"`
		EndDate      string   `json:"end_date"`
		Limit        int      `json:"limit"`
		Offset       int      `json:"offset"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Set defaults
	if request.Limit == 0 {
		request.Limit = 100
	}

	// Build dynamic query
	query := `
		SELECT
			tender_id,
			nama_paket,
			nilai_pagu,
			metode_pengadaan,
			tahun_anggaran,
			status_tender,
			tanggal_pengumuman,
			lokasi,
			kategori
		FROM nessie_iceberg.tender_data
		WHERE 1=1
	`

	// Add filters
	if request.Keyword != "" {
		query += fmt.Sprintf(" AND LOWER(nama_paket) LIKE LOWER('%%%s%%')", request.Keyword)
	}

	if request.MinValue > 0 {
		query += fmt.Sprintf(" AND nilai_pagu >= %.2f", request.MinValue)
	}

	if request.MaxValue > 0 {
		query += fmt.Sprintf(" AND nilai_pagu <= %.2f", request.MaxValue)
	}

	if len(request.Status) > 0 {
		statusList := "'" + strings.Join(request.Status, "','") + "'"
		query += fmt.Sprintf(" AND status_tender IN (%s)", statusList)
	}

	if len(request.Kategori) > 0 {
		kategoriList := "'" + strings.Join(request.Kategori, "','") + "'"
		query += fmt.Sprintf(" AND kategori IN (%s)", kategoriList)
	}

	if request.TahunAnggaran > 0 {
		query += fmt.Sprintf(" AND tahun_anggaran = %d", request.TahunAnggaran)
	}

	if request.Lokasi != "" {
		query += fmt.Sprintf(" AND LOWER(lokasi) LIKE LOWER('%%%s%%')", request.Lokasi)
	}

	if request.StartDate != "" {
		query += fmt.Sprintf(" AND tanggal_pengumuman >= '%s'", request.StartDate)
	}

	if request.EndDate != "" {
		query += fmt.Sprintf(" AND tanggal_pengumuman <= '%s'", request.EndDate)
	}

	// Add sorting and pagination
	query += fmt.Sprintf(" ORDER BY tanggal_pengumuman DESC LIMIT %d OFFSET %d",
		request.Limit, request.Offset)

	// Execute query
	results, err := h.dremio.Query(c.Request.Context(), query)
	if err != nil {
		h.logger.Error("Search query failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Search query failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   results,
		"count":  len(results),
		"limit":  request.Limit,
		"offset": request.Offset,
		"filters": request,
	})
}

// joinStrings helper for the handler
func joinStrings(elems []string, sep string) string {
	return strings.Join(elems, sep)
}