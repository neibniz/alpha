package console

import (
	"sync"
	"testing"

	"github.com/neibniz/alpha/server/internal/types"
	"gorm.io/gorm/schema"
)

func TestStatisticsResponseTypesCanBeParsedByGorm(t *testing.T) {
	testCases := []struct {
		name  string
		value any
	}{
		{name: "orders", value: &types.OrdersStatistics{}},
		{name: "users", value: &types.UserStatistics{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := schema.Parse(tc.value, &sync.Map{}, schema.NamingStrategy{})
			if err != nil {
				t.Fatalf("parse statistics response type: %v", err)
			}
			if _, ok := parsed.Relationships.Relations["List"]; ok {
				t.Fatal("List must not be parsed as a GORM relation")
			}
			if field := parsed.FieldsByName["List"]; field != nil && field.Readable {
				t.Fatal("List must not be readable during GORM scan")
			}
		})
	}
}
