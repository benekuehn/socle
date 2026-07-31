package gh

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/v71/github"
	"github.com/stretchr/testify/require"
)

func TestFindOpenPullRequestForBranchRejectsDuplicates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("per_page") != "2" {
			http.Error(w, "unexpected page size", http.StatusBadRequest)
			return
		}
		_, _ = fmt.Fprint(w, `[{"number":1},{"number":2}]`)
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	client := github.NewClient(nil)
	client.BaseURL = baseURL
	_, err = (&Client{gh: client, Owner: "owner", Repo: "repo", Ctx: context.Background()}).FindOpenPullRequestForBranch("feature")
	require.ErrorContains(t, err, "multiple open pull requests")
}
