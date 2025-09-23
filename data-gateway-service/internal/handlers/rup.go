package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"go-data-gateway/internal/clients"
)

type RUPHandler struct {
	bigquery *clients.BigQueryClient
	logger   *zap.Logger
}

func NewRUPHandler(bigquery *clients.BigQueryClient, logger *zap.Logger) *RUPHandler {
	return &RUPHandler{
		bigquery: bigquery,
		logger:   logger,
	}
}

func (h *RUPHandler) List(c *gin.Context) {
	if h.bigquery == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "BigQuery client not initialized",
		})
		return
	}

	query := `
		SELECT
			kd_rup,
			nama_paket,
			pagu,
			tahun,
			kd_satker,
			created_date
		FROM ` + "`your-dataset.rup_table`" + `
		ORDER BY created_date DESC
		LIMIT 100
	`

	results, err := h.bigquery.Query(c.Request.Context(), query)
	if err != nil {
		h.logger.Error("Failed to query RUP", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch RUP data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  results,
		"count": len(results),
	})
}

func (h *RUPHandler) GetByID(c *gin.Context) {
	if h.bigquery == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "BigQuery client not initialized",
		})
		return
	}

	id := c.Param("id")

	query := `
		SELECT *
		FROM ` + "`your-dataset.rup_table`" + `
		WHERE kd_rup = '` + id + `'
		LIMIT 1
	`

	results, err := h.bigquery.Query(c.Request.Context(), query)
	if err != nil {
		h.logger.Error("Failed to query RUP by ID", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch RUP data",
		})
		return
	}

	if len(results) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "RUP not found",
		})
		return
	}

	c.JSON(http.StatusOK, results[0])
}
