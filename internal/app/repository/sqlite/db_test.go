package sqlite

import (
    "fmt"
    "log"
    "testing"
)

func TestGetConnection(t *testing.T) {
    tests := []struct {
        name    string
        wantErr bool
    }{
        {
            name:    "getSqliteConn",
            wantErr: false,
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            db, err := GetConnection()
            defer db.Close()
            if (err != nil) != tt.wantErr {
                t.Errorf("GetConnection() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            var createTableSQL string
            err = db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='transcriptions';").Scan(&createTableSQL)
            if err != nil {
                log.Fatal(err)
            }
            fmt.Println("Create table SQL:", createTableSQL)
        })
    }
}
