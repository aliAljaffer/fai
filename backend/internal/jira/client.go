package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type Client struct {
	baseURL  string
	email    string
	apiToken string
	http     *http.Client
}

// New returns nil if Jira is unconfigured, so callers can treat it as an
// optional integration rather than a hard dependency.
func New(baseURL, email, apiToken string) *Client {
	if baseURL == "" || email == "" || apiToken == "" {
		return nil
	}
	return &Client{baseURL: baseURL, email: email, apiToken: apiToken, http: &http.Client{}}
}

type Issue struct {
	Key     string
	Summary string
	Status  string
}

type searchResponse struct {
	Issues []struct {
		Key    string `json:"key"`
		Fields struct {
			Summary string `json:"summary"`
			Status  struct {
				Name string `json:"name"`
			} `json:"status"`
		} `json:"fields"`
	} `json:"issues"`
	NextPageToken string `json:"nextPageToken"`
	IsLast        bool   `json:"isLast"`
}

// SearchIssues returns every matching issue, following Jira's nextPageToken
// pagination. maxResults is the page size (Jira caps it at 100).
func (c *Client) SearchIssues(jql string, maxResults int) ([]Issue, error) {
	if maxResults <= 0 || maxResults > 100 {
		maxResults = 100
	}
	var issues []Issue
	pageToken := ""
	seenTokens := map[string]bool{}
	for {
		page, next, isLast, err := c.searchIssuesPage(jql, maxResults, pageToken)
		if err != nil {
			return nil, err
		}
		issues = append(issues, page...)
		if isLast || next == "" || seenTokens[next] {
			break
		}
		seenTokens[next] = true
		pageToken = next
	}
	return issues, nil
}

func (c *Client) searchIssuesPage(jql string, maxResults int, pageToken string) ([]Issue, string, bool, error) {
	params := url.Values{
		"jql":        {jql},
		"maxResults": {fmt.Sprint(maxResults)},
		"fields":     {"summary,status"},
	}
	if pageToken != "" {
		params.Set("nextPageToken", pageToken)
	}
	u := c.baseURL + "/rest/api/3/search/jql?" + params.Encode()

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, "", false, err
	}
	req.SetBasicAuth(c.email, c.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, "", false, fmt.Errorf("jira: search returned status %d", resp.StatusCode)
	}

	var parsed searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, "", false, err
	}

	issues := make([]Issue, 0, len(parsed.Issues))
	for _, i := range parsed.Issues {
		issues = append(issues, Issue{Key: i.Key, Summary: i.Fields.Summary, Status: i.Fields.Status.Name})
	}
	return issues, parsed.NextPageToken, parsed.IsLast, nil
}

// GetIssue fetches a single issue by key, used to validate a manual link (phase 10).
func (c *Client) GetIssue(key string) (*Issue, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/rest/api/3/issue/"+url.PathEscape(key)+"?fields=summary,status", nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.email, c.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("jira: get issue returned status %d", resp.StatusCode)
	}

	var parsed struct {
		Key    string `json:"key"`
		Fields struct {
			Summary string `json:"summary"`
			Status  struct {
				Name string `json:"name"`
			} `json:"status"`
		} `json:"fields"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	return &Issue{Key: parsed.Key, Summary: parsed.Fields.Summary, Status: parsed.Fields.Status.Name}, nil
}

func (c *Client) IssueURL(key string) string {
	return c.baseURL + "/browse/" + key
}

// SearchByText runs a JQL "text ~" search, optionally scoped to a project, returning
// raw candidates for the caller to rank (see internal/httpapi/jira.go's fuzzy-suggest path).
func (c *Client) SearchByText(query, projectKey string, maxResults int) ([]Issue, error) {
	jql := fmt.Sprintf(`text ~ %q`, query)
	if projectKey != "" {
		jql = fmt.Sprintf(`project = %q AND %s`, projectKey, jql)
	}
	issues, _, _, err := c.searchIssuesPage(jql, maxResults, "")
	return issues, err
}

// CreateIssue creates a new issue of type "Task" in the given project and returns its key.
func (c *Client) CreateIssue(projectKey, summary string) (string, error) {
	body, err := json.Marshal(map[string]any{
		"fields": map[string]any{
			"project":   map[string]string{"key": projectKey},
			"summary":   summary,
			"issuetype": map[string]string{"name": "Task"},
		},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/rest/api/3/issue", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(c.email, c.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("jira: create issue returned status %d", resp.StatusCode)
	}

	var parsed struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	return parsed.Key, nil
}
