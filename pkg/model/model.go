package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// ItemType for thumbnail generation
type ItemType int

const (
	// TypeVideo video type
	TypeVideo ItemType = iota
	// TypeImage image type
	TypeImage
)

// ItemTypeValues string values
var ItemTypeValues = []string{"video", "image"}

// ParseItemType parse raw string into a ItemType
func ParseItemType(value string) (ItemType, error) {
	for i, short := range ItemTypeValues {
		if strings.EqualFold(short, value) {
			return ItemType(i), nil
		}
	}

	return TypeVideo, fmt.Errorf("invalid value `%s` for item type", value)
}

func (it ItemType) String() string {
	return ItemTypeValues[it]
}

// MarshalJSON marshals the enum as a quoted json string
func (it ItemType) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(it.String())
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON unmarshal JSOn
func (it *ItemType) UnmarshalJSON(b []byte) error {
	var strValue string
	if err := json.Unmarshal(b, &strValue); err != nil {
		return fmt.Errorf("unmarshal event type: %w", err)
	}

	value, err := ParseItemType(strValue)
	if err != nil {
		return fmt.Errorf("parse event type: %w", err)
	}

	*it = value
	return nil
}

// Request for generating stream
type Request struct {
	Input    string   `json:"input"`
	Output   string   `json:"output"`
	Scale    uint64   `json:"scale"`
	ItemType ItemType `json:"type"`
}

// NewRequest creates a new request
func NewRequest(input, output string, itemType ItemType, scale uint64) Request {
	return Request{
		Input:    input,
		Output:   output,
		Scale:    scale,
		ItemType: itemType,
	}
}
