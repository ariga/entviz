package entviz

import (
	"bytes"
	"context"
	"encoding/json"
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

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// ParseDevURL parses the devURL and returns the Ent dialect as well the Atlas driver name.
func ParseDevURL(devURL string) (string, string, error) {
	parsed, err := url.Parse(devURL)
	if err != nil {
		return "", "", err
	}
	switch strings.ToLower(parsed.Scheme) {
	case "sqlite", "sqlite3":
		return dialect.SQLite, "SQLITE", nil
	case "mysql":
		return dialect.MySQL, "MYSQL", nil
	case "postgres":
		return dialect.Postgres, "POSTGRESQL", nil
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
func Share(ctx context.Context, hclDocument []byte, driverName string, opts ...ShareOption) (string, error) {
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
	visualize, err := makeRequest[visualizeResponse](ctx, shareOpts.httpClient, shareOpts.endpoint, gqlRequest{
		Query: visualizeMutation,
		Variables: map[string]any{
			"text":   string(hclDocument),
			"driver": driverName,
		},
	})
	if err != nil {
		return "", fmt.Errorf("visualize request: %w", err)
	}
	share, err := makeRequest[shareResponse](ctx, shareOpts.httpClient, shareOpts.endpoint, gqlRequest{
		Query: shareVisualizationMutation,
		Variables: map[string]any{
			"extID": visualize.Data.Visualize.Node.ExtID,
		},
	})
	if err != nil {
		return "", fmt.Errorf("share request: %w", err)
	}
	if !share.Data.ShareVisualization.Success {
		return "", fmt.Errorf("could not share the visualization: %s", visualize.Data.Visualize.Node.ExtID)
	}
	return fmt.Sprintf("%s://%s/explore/%s", u.Scheme, u.Host, visualize.Data.Visualize.Node.ExtID), nil
}

// makeRequest makes a GraphQL request using the provided httpClient and endpoint.
func makeRequest[T any](ctx context.Context, httpClient *http.Client, endpoint string, r gqlRequest) (gqlResponse[T], error) {
	var (
		body    bytes.Buffer
		gqlResp gqlResponse[T]
	)
	if err := json.NewEncoder(&body).Encode(r); err != nil {
		return gqlResp, fmt.Errorf("encoding gqlRequest: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return gqlResp, fmt.Errorf("creating http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	resp, err := httpClient.Do(req)
	if err != nil {
		return gqlResp, fmt.Errorf("making http request: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusTooManyRequests:
		return gqlResp, fmt.Errorf("rate limited, try again in a few minutes")
	default:
		return gqlResp, fmt.Errorf("status code: %w", err)
	}
	if err = json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return gqlResp, fmt.Errorf("decoding gqlResponse: %w", err)
	}
	return gqlResp, nil
}

const (
	visualizeMutation = `mutation VisualizeMutation($text: String!, $driver: Driver!) {
  visualize(input: { text: $text, type: HCL, driver: $driver }) {
    node {
      extID
    }
  }
}
`
	shareVisualizationMutation = `mutation ShareVisualizationMutation($extID: String!) {
  shareVisualization(input: { fromID: $extID }) {
    success
  }
}
`
)

type gqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type gqlResponse[T any] struct {
	Data T `json:"data"`
}

type visualizeResponse struct {
	Visualize struct {
		Node struct {
			ExtID string `json:"extID"`
		} `json:"node"`
	} `json:"visualize"`
}

type shareResponse struct {
	ShareVisualization struct {
		Success bool `json:"success"`
	} `json:"shareVisualization"`
}
