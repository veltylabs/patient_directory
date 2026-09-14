package migrate

import (
	"webtyp.com/ddl"

	patientdirectory "github.com/veltylabs/patient_directory"
)

// Migrate reconcilia el esquema de base de datos del cual patient_directory es propietario: Patient.
//
// Deliberadamente NO es llamado por New, y vive en su propio paquete en lugar de un
// archivo en el paquete raíz: nada en la ruta de compilación WASM de la aplicación
// consumidora (su view.go, que importa el paquete raíz para NewView) importa jamás
// ".../patient_directory/migrate" —de modo que webtyp.com/ddl nunca entra en ese grafo
// de compilación, independientemente de los build tags de la aplicación consumidora.
//
// Sync, no CreateTable: CreateTable compila a CREATE TABLE IF NOT EXISTS y no realiza
// acción contra una tabla que ya existe, por lo que una columna agregada al modelo
// posteriormente nunca llegaría a una base de datos desplegada. Sync crea la tabla
// cuando está ausente y agrega las columnas faltantes cuando no lo está —aditivo únicamente,
// sin eliminar ni reducir, por lo que es seguro de ejecutar repetidamente.
//
// conn es un ddl.Execer, no un *orm.DB, por lo que un transporte de despliegue que solo
// puede ejecutar DDL lo satisface:
//
//	conn, _ := postgres.Open(dsn)
//	compiler, _ := conn.(ddl.Compiler)
//	err := migrate.Migrate(conn, compiler)
func Migrate(conn ddl.Execer, ddlCompiler ddl.Compiler) error {
	return ddl.New(conn, ddlCompiler).Sync(&patientdirectory.Patient{})
}
