package integration_test

import (
	"context"
	"testing"

	"ariga.io/entviz/internal/entviz"
	"entgo.io/ent/dialect"
	"github.com/stretchr/testify/require"
)

const testPostgresHCL = `table "users" {
  schema = schema.public
  column "id" {
    null = false
    type = bigint
    identity {
      generated = BY_DEFAULT
    }
  }
  column "name" {
    null = false
    type = character_varying
  }
  primary_key {
    columns = [column.id]
  }
}
schema "public" {
}
`

const testMySQLHCL = `table "users" {
  schema  = schema.dev
  collate = "utf8mb4_bin"
  column "id" {
    null           = false
    type           = bigint
    auto_increment = true
  }
  column "name" {
    null = false
    type = varchar(255)
  }
  primary_key {
    columns = [column.id]
  }
}
schema "dev" {
  charset = "utf8mb4"
  collate = "utf8mb4_0900_ai_ci"
}
`

func TestIntegrationPostgres(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	hclDocument, err := entviz.GenerateHCLFromEntSchema(ctx, entviz.GenerateOptions{
		SchemaPath:     "../entviz/testdata/ent/schema",
		Dialect:        dialect.Postgres,
		DevURL:         "postgres://postgres:pass@localhost:5432/dev?sslmode=disable",
		GlobalUniqueID: false,
	})
	require.NoError(t, err)
	require.Equal(t, testPostgresHCL, string(hclDocument))
}

func TestIntegrationMySQL(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	hclDocument, err := entviz.GenerateHCLFromEntSchema(ctx, entviz.GenerateOptions{
		SchemaPath:     "../entviz/testdata/ent/schema",
		Dialect:        dialect.MySQL,
		DevURL:         "mysql://root:pass@localhost:3306/dev",
		GlobalUniqueID: false,
	})
	require.NoError(t, err)
	require.Equal(t, testMySQLHCL, string(hclDocument))
}