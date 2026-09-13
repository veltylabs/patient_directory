package migrate_test

import (
	"testing"

	"webtyp.com/ddl"
	"webtyp.com/model"

	"github.com/veltylabs/patient_directory/migrate"
)

type dummyExecer struct {
	queries []string
}

func (e *dummyExecer) Exec(query string, args ...any) error {
	e.queries = append(e.queries, query)
	return nil
}

type dummyCompiler struct{}

func (c *dummyCompiler) CompileDDL(s ddl.Stmt, m model.Model) (string, []any, error) {
	return "CREATE TABLE IF NOT EXISTS patient (id TEXT PRIMARY KEY)", nil, nil
}

func TestMigrate(t *testing.T) {
	execer := &dummyExecer{}
	compiler := &dummyCompiler{}

	err := migrate.Migrate(execer, compiler)
	if err != nil {
		t.Fatalf("Migrate failed: %v", err)
	}

	if len(execer.queries) == 0 {
		t.Fatalf("expected queries to be executed, got 0")
	}
}
