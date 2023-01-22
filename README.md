# entviz
Visualize Ent schemas with beautiful ERDs on atlasgo.cloud

## Usage

```shell
go run -mod=mod ariga.io/entviz --help
```

```shell
Usage of ariga.io/entviz
        go run -mod=mod ariga.io/entviz <Ent schema path>
Flags:
  -dev-url string
        dev database to be used to generate the schema (default "sqlite3://file?mode=memory&cache=shared&_fk=1")
  -global-unique-id
        enable the Global Unique ID feature
```

## Example

1. Share Ent schema using `SQLite` dev database.
```shell
go run -mod=mod ariga.io/entviz ./ent/schema
```

2. Share Ent schema using `MySQL` dev database.
```shell
go run -mod=mod ariga.io/entviz -dev-url "mysql://user:pass@localhost:3306/database" ./ent/schema
```

3. Share Ent schema using `Postgres` dev database.
```shell
go run -mod=mod ariga.io/entviz -dev-url "postgres://postgres:pass@localhost:5432/database?sslmode=disable" ./ent/schema
```