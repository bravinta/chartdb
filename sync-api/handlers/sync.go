package handlers

import (
	"sync-api/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// SyncHandler provides handlers for sync operations
type SyncHandler struct {
	DB *gorm.DB
}

// NewSyncHandler creates a new SyncHandler
func NewSyncHandler(db *gorm.DB) *SyncHandler {
	return &SyncHandler{DB: db}
}

// PullResponse represents the response for the pull endpoint
type PullResponse struct {
	Diagram       *models.Diagram         `json:"diagram"`
	Tables        []models.DBTable        `json:"tables"`
	Relationships []models.DBRelationship `json:"relationships"`
	Dependencies  []models.DBDependency   `json:"dependencies"`
}

// Pull retrieves a diagram and its associated data
func (h *SyncHandler) Pull(c *fiber.Ctx) error {
	diagramID := c.Params("diagramId")
	if diagramID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Diagram ID is required",
		})
	}

	var diagram models.Diagram
	if err := h.DB.First(&diagram, "id = ?", diagramID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Diagram not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve diagram",
		})
	}

	var tables []models.DBTable
	if err := h.DB.Where("diagram_id = ?", diagramID).Find(&tables).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve tables",
		})
	}

	var relationships []models.DBRelationship
	if err := h.DB.Where("diagram_id = ?", diagramID).Find(&relationships).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve relationships",
		})
	}

	var dependencies []models.DBDependency
	if err := h.DB.Where("diagram_id = ?", diagramID).Find(&dependencies).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve dependencies",
		})
	}

	// Make sure we don't return null for slices if they are empty
	if tables == nil {
		tables = []models.DBTable{}
	}
	if relationships == nil {
		relationships = []models.DBRelationship{}
	}
	if dependencies == nil {
		dependencies = []models.DBDependency{}
	}

	response := PullResponse{
		Diagram:       &diagram,
		Tables:        tables,
		Relationships: relationships,
		Dependencies:  dependencies,
	}

	return c.JSON(response)
}
