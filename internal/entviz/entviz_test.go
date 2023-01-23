package entviz

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
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

func Test_Share(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	t.Run("normal flow", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, userAgent, r.UserAgent())
			require.Equal(t, "application/json", r.Header.Get("Content-Type"))
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")

			if bytes.Contains(body, []byte(`VisualizeMutation`)) {
				require.Equal(t, visualizeQueryRequest, string(body))
				_, _ = w.Write([]byte(`{"data":{"visualize":{"node":{"extID":"23098224"}}}}`))
			}
			if bytes.Contains(body, []byte(`ShareVisualizationMutation`)) {
				require.Equal(t, shareQueryRequest, string(body))
				_, _ = w.Write([]byte(`{"data":{"shareVisualization":{"success":true}}}`))
			}
		}))
		defer srv.Close()
		link, err := Share(ctx, []byte(testHCL), dialect.SQLite,
			ShareWithHttpClient(&http.Client{}),
			ShareWithEndpoint(srv.URL),
		)
		require.NoError(t, err)
		require.Equal(t, srv.URL+"/explore/23098224", link)
	})
	t.Run("rate limited", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, userAgent, r.UserAgent())
			require.Equal(t, "application/json", r.Header.Get("Content-Type"))
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer srv.Close()
		_, err := Share(ctx, []byte(testHCL), dialect.SQLite,
			ShareWithHttpClient(&http.Client{}),
			ShareWithEndpoint(srv.URL),
		)
		require.ErrorContains(t, err, "visualize request: returned error 429 Too Many Requests: ")
	})
}

const (
	shareQueryRequest     = "{\"query\":\"\\nmutation ShareVisualizationMutation ($extID: String!) {\\n\\tshareVisualization(input: {fromID:$extID}) {\\n\\t\\tsuccess\\n\\t}\\n}\\n\",\"variables\":{\"extID\":\"23098224\"},\"operationName\":\"ShareVisualizationMutation\"}"
	visualizeQueryRequest = "{\"query\":\"\\nmutation VisualizeMutation ($text: String!, $driver: Driver!) {\\n\\tvisualize(input: {text:$text,type:HCL,driver:$driver}) {\\n\\t\\tnode {\\n\\t\\t\\textID\\n\\t\\t}\\n\\t}\\n}\\n\",\"variables\":{\"text\":\"table \\\"users\\\" {\\n  schema = schema.main\\n  column \\\"id\\\" {\\n    null           = false\\n    type           = integer\\n    auto_increment = true\\n  }\\n  column \\\"name\\\" {\\n    null = false\\n    type = text\\n  }\\n  primary_key {\\n    columns = [column.id]\\n  }\\n}\\nschema \\\"main\\\" {\\n}\\n\",\"driver\":\"sqlite3\"},\"operationName\":\"VisualizeMutation\"}"
)
