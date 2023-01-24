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

Share Ent schema using `SQLite` dev database.

```shell
❯ go run -mod=mod ariga.io/entviz ./ent/schema
Here is a public link to your schema visualization
        https://gh.atlasgo.cloud/explore/c3aa3f24
```

For `MySQL` or `Postgres` check the examples below:

```shell
❯ go run -mod=mod ariga.io/entviz -dev-url "mysql://user:pass@localhost:3306/database" ./ent/schema
❯ go run -mod=mod ariga.io/entviz -dev-url "postgres://postgres:pass@localhost:5432/database?sslmode=disable" ./ent/schema
```
