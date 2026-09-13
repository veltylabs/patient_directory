package migrate

import (
	"webtyp.com/ddl"

	patientdirectory "github.com/veltylabs/patient_directory"
)

// Migrate reconciles the database schema patient_directory owns: Patient.
//
// It is deliberately NOT called by New, and deliberately lives in its own
// package rather than a file in the root package: nothing on a consuming app's
// WASM build path (its view.go, which imports the root package for NewView)
// ever imports ".../patient_directory/migrate" — so webtyp.com/ddl never
// enters that build graph, regardless of build tags on the consumer's side.
//
// Sync, not CreateTable: CreateTable compiles to CREATE TABLE IF NOT EXISTS and
// is a no-op against a table that already exists, so a column added to the
// model later would never reach a deployed database. Sync creates the table
// when it is absent and adds the missing columns when it is not — additive
// only, never dropping or narrowing, so it is safe to run repeatedly.
//
// conn is a ddl.Execer, not an *orm.DB, so a deploy-time transport that can
// only execute DDL satisfies it:
//
//	conn, _ := postgres.Open(dsn)
//	compiler, _ := conn.(ddl.Compiler)
//	err := migrate.Migrate(conn, compiler)
func Migrate(conn ddl.Execer, ddlCompiler ddl.Compiler) error {
	return ddl.New(conn, ddlCompiler).Sync(&patientdirectory.Patient{})
}
