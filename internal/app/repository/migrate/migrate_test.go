package migrate

import "testing"

func TestMigrateToPostgres(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			"migrate_legacy_data_to_pg",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			MigrateToPostgres()
		})
	}
}
