package pg

import (
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGetByFileHashFailsWhenSchemaMissingColumn(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	pdb := &PostgresDB{db: db}

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = current_schema()
		  AND table_name = 'transcriptions'`)).
		WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("id").AddRow("file_name"))

	if _, err := pdb.GetByFileHash("hash-1"); err == nil {
		t.Fatalf("expected missing file_hash column error")
	}
}

func TestUpdateFileMetadataFailsWhenSchemaMissingColumns(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	pdb := &PostgresDB{db: db}

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = current_schema()
		  AND table_name = 'transcriptions'`)).
		WillReturnRows(sqlmock.NewRows([]string{"column_name"}).AddRow("id").AddRow("provider_type"))

	if err := pdb.UpdateFileMetadata(1, "hash-1", 123); err == nil {
		t.Fatalf("expected missing file metadata columns error")
	}
}
