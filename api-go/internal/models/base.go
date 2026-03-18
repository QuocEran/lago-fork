package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// BaseModel provides UUID primary key and timestamps for all models.
// Convention: all models embed BaseModel and implement TableName() returning
// the snake_case plural table name.
type BaseModel struct {
	ID        string    `gorm:"primarykey;type:uuid;default:gen_random_uuid()"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// SoftDeleteModel extends BaseModel with GORM soft-delete support.
// Use this for any domain entity that uses deleted_at instead of hard deletes.
type SoftDeleteModel struct {
	BaseModel
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// StringArray maps a Go []string to a PostgreSQL text[] column.
// It serialises as the canonical Postgres array literal: {elem1,elem2,...}
type StringArray []string

func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return "{}", nil
	}
	quoted := make([]string, len(s))
	for i, v := range s {
		escaped := strings.ReplaceAll(v, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		quoted[i] = `"` + escaped + `"`
	}
	return "{" + strings.Join(quoted, ",") + "}", nil
}

func (s *StringArray) Scan(src any) error {
	if src == nil {
		*s = StringArray{}
		return nil
	}
	var raw string
	switch v := src.(type) {
	case string:
		raw = v
	case []byte:
		raw = string(v)
	default:
		return fmt.Errorf("StringArray: cannot scan type %T", src)
	}
	raw = strings.TrimSpace(raw)
	if raw == "{}" || raw == "" {
		*s = StringArray{}
		return nil
	}
	// Strip outer braces and decode via JSON array syntax.
	inner := strings.TrimPrefix(strings.TrimSuffix(raw, "}"), "{")
	// Wrap as JSON array for convenience.
	var decoded []string
	if err := json.Unmarshal([]byte("["+inner+"]"), &decoded); err != nil {
		// Fallback: plain comma split for simple unquoted arrays.
		parts := strings.Split(inner, ",")
		decoded = make([]string, len(parts))
		for i, p := range parts {
			decoded[i] = strings.Trim(strings.TrimSpace(p), `"`)
		}
	}
	*s = decoded
	return nil
}

// JSONBMap maps a Go map[string]any to a PostgreSQL jsonb column.
type JSONBMap map[string]any

func (j JSONBMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	b, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (j *JSONBMap) Scan(src any) error {
	if src == nil {
		*j = JSONBMap{}
		return nil
	}
	switch v := src.(type) {
	case []byte:
		return json.Unmarshal(v, j)
	case string:
		return json.Unmarshal([]byte(v), j)
	default:
		return fmt.Errorf("JSONBMap: cannot scan type %T", src)
	}
}
