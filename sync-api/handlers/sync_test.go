package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"sync-api/middleware"
	"sync-api/models"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupApp(t *testing.T) (*fiber.App, *gorm.DB) {
	os.Setenv("API_SECRET", "test-secret")

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to in-memory db: %v", err)
	}

	err = db.AutoMigrate(
		&models.Diagram{},
		&models.DBTable{},
		&models.DBRelationship{},
		&models.DBDependency{},
	)
	if err != nil {
		t.Fatalf("Failed to auto migrate database: %v", err)
	}

	app := fiber.New()
	syncHandler := NewSyncHandler(db)

	api := app.Group("/api")
	syncGroup := api.Group("/sync", middleware.AuthMiddleware())
	syncGroup.Get("/pull/:diagramId", syncHandler.Pull)
	syncGroup.Post("/push", syncHandler.Push)

	return app, db
}

func ptrString(s string) *string { return &s }

func TestPull_Unauthorized(t *testing.T) {
	app, _ := setupApp(t)

	req := httptest.NewRequest("GET", "/api/sync/pull/123", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected status 401, got %v", resp.StatusCode)
	}
}

func TestPull_NotFound(t *testing.T) {
	app, _ := setupApp(t)

	req := httptest.NewRequest("GET", "/api/sync/pull/999", nil)
	req.Header.Set("X-API-Secret", "test-secret")
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusNotFound {
		t.Errorf("Expected status 404, got %v", resp.StatusCode)
	}
}

func TestPush_SuccessAndPull_Success(t *testing.T) {
	app, _ := setupApp(t)

	// Create test data
	now := time.Now()
	diagram := models.Diagram{
		ID:           "test-diagram",
		Name:         "Test Diagram",
		DatabaseType: "PostgreSQL",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	table1 := models.DBTable{
		ID:        "table-1",
		DiagramID: "test-diagram",
		Name:      "users",
		X:         100,
		Y:         200,
		Fields:    []byte(`[{"id":"field-1","name":"id","type":"integer"}]`),
		Indexes:   []byte(`[]`),
		Color:     "#fff",
		IsView:    false,
		CreatedAt: now.UnixMilli(),
	}

	pushReq := PushRequest{
		Diagram: &diagram,
		Tables:  []models.DBTable{table1},
	}

	body, _ := json.Marshal(pushReq)
	req := httptest.NewRequest("POST", "/api/sync/push", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Secret", "test-secret")

	// 1. Test Push
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status 200 on push, got %v", resp.StatusCode)
	}

	// 2. Test Pull
	reqPull := httptest.NewRequest("GET", "/api/sync/pull/test-diagram", nil)
	reqPull.Header.Set("X-API-Secret", "test-secret")
	respPull, _ := app.Test(reqPull)

	if respPull.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status 200 on pull, got %v", respPull.StatusCode)
	}

	var pullRes PullResponse
	json.NewDecoder(respPull.Body).Decode(&pullRes)

	if pullRes.Diagram.ID != "test-diagram" {
		t.Errorf("Expected diagram ID 'test-diagram', got %v", pullRes.Diagram.ID)
	}

	if len(pullRes.Tables) != 1 {
		t.Errorf("Expected 1 table, got %v", len(pullRes.Tables))
	} else if pullRes.Tables[0].Name != "users" {
		t.Errorf("Expected table name 'users', got %v", pullRes.Tables[0].Name)
	}
}

func TestPush_LastWriteWins(t *testing.T) {
	app, db := setupApp(t)

	now := time.Now()
	oldTime := now.Add(-1 * time.Hour)
	newTime := now.Add(1 * time.Hour)

	// Initial setup in DB
	diagram := models.Diagram{
		ID:           "diagram-conflict",
		Name:         "Initial Diagram",
		DatabaseType: "PostgreSQL",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	db.Create(&diagram)

	table := models.DBTable{
		ID:        "table-conflict",
		DiagramID: "diagram-conflict",
		Name:      "initial_table",
		X:         0,
		Y:         0,
		Fields:    []byte(`[]`),
		Indexes:   []byte(`[]`),
		Color:     "#fff",
		CreatedAt: now.UnixMilli(),
	}
	db.Create(&table)

	// 1. Attempt to push OLDER data (should be ignored)
	oldDiagram := diagram
	oldDiagram.Name = "Old Diagram"
	oldDiagram.UpdatedAt = oldTime

	oldTable := table
	oldTable.Name = "old_table"
	oldTable.CreatedAt = oldTime.UnixMilli()

	pushOldReq := PushRequest{
		Diagram: &oldDiagram,
		Tables:  []models.DBTable{oldTable},
	}

	bodyOld, _ := json.Marshal(pushOldReq)
	reqOld := httptest.NewRequest("POST", "/api/sync/push", bytes.NewReader(bodyOld))
	reqOld.Header.Set("Content-Type", "application/json")
	reqOld.Header.Set("X-API-Secret", "test-secret")
	respOld, _ := app.Test(reqOld)

	if respOld.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status 200 on old push, got %v", respOld.StatusCode)
	}

	// Verify DB didn't change
	var currentDiag models.Diagram
	db.First(&currentDiag, "id = ?", "diagram-conflict")
	if currentDiag.Name != "Initial Diagram" {
		t.Errorf("Expected diagram name to remain 'Initial Diagram', got %v", currentDiag.Name)
	}

	var currentTable models.DBTable
	db.First(&currentTable, "id = ?", "table-conflict")
	if currentTable.Name != "initial_table" {
		t.Errorf("Expected table name to remain 'initial_table', got %v", currentTable.Name)
	}

	// 2. Attempt to push NEWER data (should update)
	newDiagram := diagram
	newDiagram.Name = "New Diagram"
	newDiagram.UpdatedAt = newTime

	newTable := table
	newTable.Name = "new_table"
	newTable.CreatedAt = newTime.UnixMilli()

	pushNewReq := PushRequest{
		Diagram: &newDiagram,
		Tables:  []models.DBTable{newTable},
	}

	bodyNew, _ := json.Marshal(pushNewReq)
	reqNew := httptest.NewRequest("POST", "/api/sync/push", bytes.NewReader(bodyNew))
	reqNew.Header.Set("Content-Type", "application/json")
	reqNew.Header.Set("X-API-Secret", "test-secret")
	respNew, _ := app.Test(reqNew)

	if respNew.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status 200 on new push, got %v", respNew.StatusCode)
	}

	// Verify DB updated
	db.First(&currentDiag, "id = ?", "diagram-conflict")
	if currentDiag.Name != "New Diagram" {
		t.Errorf("Expected diagram name to be 'New Diagram', got %v", currentDiag.Name)
	}

	db.First(&currentTable, "id = ?", "table-conflict")
	if currentTable.Name != "new_table" {
		t.Errorf("Expected table name to be 'new_table', got %v", currentTable.Name)
	}
}

func TestPush_MissingDiagram(t *testing.T) {
	app, _ := setupApp(t)

	pushReq := PushRequest{
		Diagram: nil,
	}

	body, _ := json.Marshal(pushReq)
	req := httptest.NewRequest("POST", "/api/sync/push", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Secret", "test-secret")

	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected status 400 when missing diagram, got %v", resp.StatusCode)
	}
}
