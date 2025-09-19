package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"go-data-gateway/internal/clients"
)

type QueryHandler struct {
	dremio   *clients.DremioClient
	bigquery *clients.BigQueryClient
	logger   *zap.Logger
}

func NewQueryHandler(dremio *clients.DremioClient, bigquery *clients.BigQueryClient, logger *zap.Logger) *QueryHandler {
	return &QueryHandler{
		dremio:   dremio,
		bigquery: bigquery,
		logger:   logger,
	}
}

type QueryRequest struct {
	SQL    string `json:"sql" binding:"required"`
	Source string `json:"source" binding:"required,oneof=dremio bigquery"`
}

func (h *QueryHandler) Execute(c *gin.Context) {
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	h.logger.Info("Executing query",
		zap.String("source", req.Source),
		zap.String("sql", req.SQL))

	var result interface{}
	var err error

	switch req.Source {
	case "dremio":
		if h.dremio == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Dremio client not initialized",
			})
			return
		}
		result, err = h.dremio.ExecuteQuery(c.Request.Context(), req.SQL)

	case "bigquery":
		if h.bigquery == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "BigQuery client not initialized",
			})
			return
		}
		result, err = h.bigquery.ExecuteQuery(c.Request.Context(), req.SQL)

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid source",
		})
		return
	}

	if err != nil {
		h.logger.Error("Query execution failed",
			zap.String("source", req.Source),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, result)
}