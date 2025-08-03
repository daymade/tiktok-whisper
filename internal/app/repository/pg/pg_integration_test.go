//go:build integration
// +build integration

package pg

import (
	"database/sql"
	"fmt"
	"log"
	"testing"
)

func TestGetConnection_Integration(t *testing.T) {
	tests := []struct {
		name    string
		want    *sql.DB
		wantErr bool
	}{
		{
			name:    "getPgConn",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := GetConnection()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConnection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			var createTableSQL string
			err = db.QueryRow("SELECT show_create_table('public', 'transcriptions');").Scan(&createTableSQL)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("Create table SQL:", createTableSQL)

		})
	}
}