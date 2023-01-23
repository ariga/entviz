package entviz

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	_ "ariga.io/atlas/sql/mysql"
	_ "ariga.io/atlas/sql/postgres"
	atlas "ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlclient"
	_ "ariga.io/atlas/sql/sqlite"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql/schema"
	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"github.com/Khan/genqlient/graphql"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// ParseDevURL parses the devURL and returns the Ent dialect as well the Atlas driver name.
func ParseDevURL(devURL string) (string, Driver, error) {
	parsed, err := url.Parse(devURL)
	if err != nil {
		return "", "", err
	}
	switch strings.ToLower(parsed.Scheme) {
	case "sqlite", "sqlite3":
		return dialect.SQLite, DriverSqlite, nil
	case "mysql":
		return dialect.MySQL, DriverMysql, nil
	case "postgres":
		return dialect.Postgres, DriverPostgresql, nil
	}
	return "", "", fmt.Errorf("unknow dialect: %s", parsed.Scheme)
}

var (
	errSkip = errors.New("skip")
)

// HCLOptions are the options that can be provided to HCL.
type HCLOptions struct {
	SchemaPath     string
	Dialect        string
	DevURL         string
	GlobalUniqueID bool
}

// HCL generates an Atlas HCL document from an Ent schema.
// Most of the code below is taken from https://github.com/rotemtam/entprint.
func HCL(ctx context.Context, hclOpts HCLOptions) ([]byte, error) {
	graph, err := entc.LoadGraph(hclOpts.SchemaPath, &gen.Config{})
	if err != nil {
		return nil, fmt.Errorf("loading schema: %w", err)
	}
	var sch *atlas.Schema
	opts := []schema.MigrateOption{
		schema.WithGlobalUniqueID(hclOpts.GlobalUniqueID),
		schema.WithDiffHook(func(differ schema.Differ) schema.Differ {
			return schema.DiffFunc(func(current, desired *atlas.Schema) ([]atlas.Change, error) {
				sch = desired
				return nil, errSkip
			})
		}),
		schema.WithDialect(hclOpts.Dialect),
	}
	mig, err := schema.NewMigrateURL(hclOpts.DevURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("creating migration engine: %w", err)
	}
	tbl, err := graph.Tables()
	if err != nil {
		return nil, fmt.Errorf("reading tables: %w", err)
	}
	if err := mig.Create(ctx, tbl...); err != nil && !errors.Is(err, errSkip) {
		return nil, fmt.Errorf("creating schema: %w", err)
	}
	drv, err := sqlclient.Open(ctx, hclOpts.DevURL)
	if err != nil {
		return nil, fmt.Errorf("opening sql client: %w", err)
	}
	norm, ok := drv.Driver.(atlas.Normalizer)
	if ok {
		sch, err = norm.NormalizeSchema(ctx, sch)
		if err != nil {
			return nil, fmt.Errorf("normalizing schema: %w", err)
		}
	}
	spec, err := drv.MarshalSpec(sch)
	if err != nil {
		return nil, fmt.Errorf("marshaling schema: %w", err)
	}
	return spec, nil
}

const userAgent = "EntViz"

type shareOpts struct {
	endpoint   string
	httpClient *http.Client
}

type ShareOption func(opts *shareOpts)

// ShareWithEndpoint allows providing a custom endpoint to shareHCL.
func ShareWithEndpoint(endpoint string) ShareOption {
	return func(opts *shareOpts) {
		opts.endpoint = endpoint
	}
}

// ShareWithHttpClient allows proving a custom *http.Client for shareHCL.
func ShareWithHttpClient(httpClient *http.Client) ShareOption {
	return func(opts *shareOpts) {
		opts.httpClient = httpClient
	}
}

// Share create and returns an Atlas Cloud Explore link for the given HCL document.
func Share(ctx context.Context, hclDocument []byte, driver Driver, opts ...ShareOption) (string, error) {
	shareOpts := shareOpts{
		endpoint:   "https://gh.atlasgo.cloud/api/query",
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
	for _, opt := range opts {
		opt(&shareOpts)
	}
	u, err := url.Parse(shareOpts.endpoint)
	if err != nil {
		return "", fmt.Errorf("parsing endpoint: %w", err)
	}
	gql := graphql.NewClient(shareOpts.endpoint, &entvizDo{httpClient: shareOpts.httpClient})
	visualize, err := VisualizeMutation(ctx, gql, string(hclDocument), driver)
	if err != nil {
		return "", fmt.Errorf("visualize request: %w", err)
	}
	share, err := ShareVisualizationMutation(ctx, gql, visualize.Visualize.Node.ExtID)
	if err != nil {
		return "", fmt.Errorf("share request: %w", err)
	}
	if !share.ShareVisualization.Success {
		return "", fmt.Errorf("could not share the visualization: %s", visualize.Visualize.Node.ExtID)
	}
	return fmt.Sprintf("%s://%s/explore/%s", u.Scheme, u.Host, visualize.Visualize.Node.ExtID), nil
}

// entvizDo implements graphql.Doer and sets
// the userAgent on each request.
type entvizDo struct {
	httpClient *http.Client
}

func (e *entvizDo) Do(r *http.Request) (*http.Response, error) {
	r.Header.Set("User-Agent", userAgent)
	return e.httpClient.Do(r)
}
