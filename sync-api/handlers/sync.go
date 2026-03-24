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

// PushRequest represents the request for the push endpoint
type PushRequest struct {
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

// Push receives a diagram and its associated data and updates the database
// using a Last-Write-Wins strategy for conflict resolution.
func (h *SyncHandler) Push(c *fiber.Ctx) error {
	var req PushRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}

	if req.Diagram == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Diagram is required",
		})
	}

	// Start a transaction
	tx := h.DB.Begin()
	if tx.Error != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to start transaction",
		})
	}

	// Helper to handle rollback and response
	handleError := func(err string, status int) error {
		tx.Rollback()
		return c.Status(status).JSON(fiber.Map{
			"error": err,
		})
	}

	// Process Diagram
	var existingDiagram models.Diagram
	err := tx.First(&existingDiagram, "id = ?", req.Diagram.ID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Insert
			if err := tx.Create(req.Diagram).Error; err != nil {
				return handleError("Failed to create diagram", fiber.StatusInternalServerError)
			}
		} else {
			return handleError("Failed to retrieve diagram", fiber.StatusInternalServerError)
		}
	} else {
		// Update if Last-Write-Wins (using UpdatedAt)
		if req.Diagram.UpdatedAt.After(existingDiagram.UpdatedAt) {
			if err := tx.Save(req.Diagram).Error; err != nil {
				return handleError("Failed to update diagram", fiber.StatusInternalServerError)
			}
		}
	}

	// Process Tables
	for _, table := range req.Tables {
		var existingTable models.DBTable
		err := tx.First(&existingTable, "id = ?", table.ID).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := tx.Create(&table).Error; err != nil {
					return handleError("Failed to create table", fiber.StatusInternalServerError)
				}
			} else {
				return handleError("Failed to retrieve table", fiber.StatusInternalServerError)
			}
		} else {
			// Update if Last-Write-Wins (using CreatedAt)
			if table.CreatedAt > existingTable.CreatedAt {
				if err := tx.Save(&table).Error; err != nil {
					return handleError("Failed to update table", fiber.StatusInternalServerError)
				}
			}
		}
	}

	// Process Relationships
	for _, rel := range req.Relationships {
		var existingRel models.DBRelationship
		err := tx.First(&existingRel, "id = ?", rel.ID).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := tx.Create(&rel).Error; err != nil {
					return handleError("Failed to create relationship", fiber.StatusInternalServerError)
				}
			} else {
				return handleError("Failed to retrieve relationship", fiber.StatusInternalServerError)
			}
		} else {
			// Update if Last-Write-Wins (using CreatedAt)
			if rel.CreatedAt > existingRel.CreatedAt {
				if err := tx.Save(&rel).Error; err != nil {
					return handleError("Failed to update relationship", fiber.StatusInternalServerError)
				}
			}
		}
	}

	// Process Dependencies
	for _, dep := range req.Dependencies {
		var existingDep models.DBDependency
		err := tx.First(&existingDep, "id = ?", dep.ID).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := tx.Create(&dep).Error; err != nil {
					return handleError("Failed to create dependency", fiber.StatusInternalServerError)
				}
			} else {
				return handleError("Failed to retrieve dependency", fiber.StatusInternalServerError)
			}
		} else {
			// Update if Last-Write-Wins (using CreatedAt)
			if dep.CreatedAt > existingDep.CreatedAt {
				if err := tx.Save(&dep).Error; err != nil {
					return handleError("Failed to update dependency", fiber.StatusInternalServerError)
				}
			}
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to commit transaction",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}
