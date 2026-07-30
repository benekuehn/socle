package gh

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/v83/github"
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

func TestFindPullRequestForBranchIncludesClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/repos/owner/repo/pulls", r.URL.Path)
		require.Equal(t, "all", r.URL.Query().Get("state"))
		require.Equal(t, "owner:feature", r.URL.Query().Get("head"))
		_, _ = fmt.Fprint(w, `[{"number":42,"state":"closed"}]`)
	}))
	defer server.Close()

	baseURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)
	client := github.NewClient(nil)
	client.BaseURL = baseURL
	pr, err := (&Client{gh: client, Owner: "owner", Repo: "repo", Ctx: context.Background()}).FindPullRequestForBranch("feature")
	require.NoError(t, err)
	require.Equal(t, 42, pr.GetNumber())
}
