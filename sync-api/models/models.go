package models

import (
	"encoding/json"
	"time"
)

// Diagram represents a database diagram.
type Diagram struct {
	ID              string    `gorm:"primaryKey;type:varchar(255)" json:"id"`
	Name            string    `json:"name"`
	DatabaseType    string    `json:"databaseType"`
	DatabaseEdition *string   `json:"databaseEdition,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// DBTable represents a table in the database diagram.
type DBTable struct {
	ID                 string          `gorm:"primaryKey;type:varchar(255)" json:"id"`
	DiagramID          string          `gorm:"type:varchar(255);index;not null" json:"diagramId"`
	Name               string          `json:"name"`
	Schema             *string         `json:"schema,omitempty"`
	X                  float64         `json:"x"`
	Y                  float64         `json:"y"`
	Fields             json.RawMessage `gorm:"type:jsonb" json:"fields"`
	Indexes            json.RawMessage `gorm:"type:jsonb" json:"indexes"`
	CheckConstraints   json.RawMessage `gorm:"type:jsonb" json:"checkConstraints,omitempty"`
	Color              string          `json:"color"`
	IsView             bool            `json:"isView"`
	IsMaterializedView *bool           `json:"isMaterializedView,omitempty"`
	CreatedAt          int64           `json:"createdAt"`
	Width              *float64        `json:"width,omitempty"`
	Comments           *string         `json:"comments,omitempty"`
	Order              *int            `json:"order,omitempty"`
	Expanded           *bool           `json:"expanded,omitempty"`
	ParentAreaID       *string         `json:"parentAreaId,omitempty"`
}

// DBRelationship represents a relationship between two tables.
type DBRelationship struct {
	ID                string  `gorm:"primaryKey;type:varchar(255)" json:"id"`
	DiagramID         string  `gorm:"type:varchar(255);index;not null" json:"diagramId"`
	Name              string  `json:"name"`
	SourceSchema      *string `json:"sourceSchema,omitempty"`
	SourceTableID     string  `json:"sourceTableId"`
	TargetSchema      *string `json:"targetSchema,omitempty"`
	TargetTableID     string  `json:"targetTableId"`
	SourceFieldID     string  `json:"sourceFieldId"`
	TargetFieldID     string  `json:"targetFieldId"`
	SourceCardinality string  `json:"sourceCardinality"`
	TargetCardinality string  `json:"targetCardinality"`
	CreatedAt         int64   `json:"createdAt"`
}

// DBDependency represents a dependency between two tables (e.g., views).
type DBDependency struct {
	ID               string  `gorm:"primaryKey;type:varchar(255)" json:"id"`
	DiagramID        string  `gorm:"type:varchar(255);index;not null" json:"diagramId"`
	Schema           *string `json:"schema,omitempty"`
	TableID          string  `json:"tableId"`
	DependentSchema  *string `json:"dependentSchema,omitempty"`
	DependentTableID string  `json:"dependentTableId"`
	CreatedAt        int64   `json:"createdAt"`
}
