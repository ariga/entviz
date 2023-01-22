package entviz

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"entgo.io/ent/dialect"
	"github.com/stretchr/testify/require"
)

const testHCL = `table "users" {
  schema = schema.main
  column "id" {
    null           = false
    type           = integer
    auto_increment = true
  }
  column "name" {
    null = false
    type = text
  }
  primary_key {
    columns = [column.id]
  }
}
schema "main" {
}
`

func Test_generateHCLFromEntSchema(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	hclDocument, err := GenerateHCLFromEntSchema(ctx, GenerateOptions{
		SchemaPath:     "./testdata/ent/schema",
		Dialect:        dialect.SQLite,
		DevURL:         "sqlite3://file?mode=memory&cache=shared&_fk=1",
		GlobalUniqueID: false,
	})
	require.NoError(t, err)
	require.Equal(t, testHCL, string(hclDocument))
}

func Test_shareHCL(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	link, err := ShareHCL(ctx, []byte(testHCL), dialect.SQLite,
		ShareWithHttpClient(&http.Client{
			Transport: roundTripFunc(func(req *http.Request) *http.Response {
				require.Equal(t, userAgent, req.UserAgent())
				require.Equal(t, "application/json", req.Header.Get("Content-Type"))
				body, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				var responseBody io.ReadCloser
				if bytes.Contains(body, []byte(`VisualizeMutation`)) {
					require.Equal(t, visualizeQueryRequest+"\n", string(body))
					responseBody = io.NopCloser(strings.NewReader(`{"data":{"visualize":{"node":{"extID":"23098224"}}}}`))
				}
				if bytes.Contains(body, []byte(`ShareVisualizationMutation`)) {
					require.Equal(t, shareQueryRequest+"\n", string(body))
					responseBody = io.NopCloser(strings.NewReader(`{"data":{"shareVisualization":{"success":true}}}`))
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     map[string][]string{"Content-Type": {"application/json"}},
					Body:       responseBody,
				}
			}),
		}),
		ShareWithEndpoint(`https://gh.ariga.cloud/api/query`),
	)
	require.NoError(t, err)
	require.Equal(t, "https://gh.ariga.cloud/explore/23098224", link)
}

const (
	shareQueryRequest     = `{"query":"mutation ShareVisualizationMutation($extID: String!) {\n  shareVisualization(input: { fromID: $extID }) {\n    success\n  }\n}\n","variables":{"extID":"23098224"}}`
	visualizeQueryRequest = `{"query":"mutation VisualizeMutation($text: String!, $driver: Driver!) {\n  visualize(input: { text: $text, type: HCL, driver: $driver }) {\n    node {\n      extID\n    }\n  }\n}\n","variables":{"driver":"sqlite3","text":"table \"users\" {\n  schema = schema.main\n  column \"id\" {\n    null           = false\n    type           = integer\n    auto_increment = true\n  }\n  column \"name\" {\n    null = false\n    type = text\n  }\n  primary_key {\n    columns = [column.id]\n  }\n}\nschema \"main\" {\n}\n"}}`
)

type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}
