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
	Key            string
	Summary        string
	Description    string
	Status         string
	IssueType      string
	ParentKey      string
	ResolutionDate string
	Created        string
	Updated        string
	AssigneeID     string
}

type Comment struct {
	ID         string
	AuthorID   string
	AuthorName string
	Body       string
	ParentID   string
}

type user struct {
	AccountID    string `json:"accountId"`
	Name         string `json:"name"`
	EmailAddress string `json:"emailAddress"`
	DisplayName  string `json:"displayName"`
}

func (u user) id() string {
	if u.AccountID != "" {
		return u.AccountID
	}
	if u.Name != "" {
		return u.Name
	}
	return u.EmailAddress
}

type searchResponse struct {
	Issues []struct {
		Key    string `json:"key"`
		Fields struct {
			Summary        string          `json:"summary"`
			Description    json.RawMessage `json:"description"`
			ResolutionDate string          `json:"resolutiondate"`
			Created        string          `json:"created"`
			Updated        string          `json:"updated"`
			Status         struct {
				Name string `json:"name"`
			} `json:"status"`
			IssueType struct {
				Name string `json:"name"`
			} `json:"issuetype"`
			Parent *struct {
				Key string `json:"key"`
			} `json:"parent"`
			Assignee *user `json:"assignee"`
		} `json:"fields"`
	} `json:"issues"`
	NextPageToken string `json:"nextPageToken"`
	IsLast        bool   `json:"isLast"`
}

// flexID accepts a Jira id sent as either a JSON string or a number; comment
// ids arrive both ways depending on the endpoint.
type flexID string

func (f *flexID) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*f = ""
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*f = flexID(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*f = flexID(n.String())
	return nil
}

type commentsResponse struct {
	StartAt    int `json:"startAt"`
	MaxResults int `json:"maxResults"`
	Total      int `json:"total"`
	Comments   []struct {
		ID       flexID          `json:"id"`
		Body     json.RawMessage `json:"body"`
		ParentID flexID          `json:"parentId"`
		Author   user            `json:"author"`
	} `json:"comments"`
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
		"fields":     {"summary,description,status,issuetype,parent,subtasks,resolutiondate,created,updated,assignee"},
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
		issue := Issue{
			Key: i.Key, Summary: i.Fields.Summary, Status: i.Fields.Status.Name,
			IssueType:      i.Fields.IssueType.Name,
			Description:    rawToMarkdown(i.Fields.Description),
			ResolutionDate: i.Fields.ResolutionDate, Created: i.Fields.Created, Updated: i.Fields.Updated,
		}
		if i.Fields.Parent != nil {
			issue.ParentKey = i.Fields.Parent.Key
		}
		if i.Fields.Assignee != nil {
			issue.AssigneeID = i.Fields.Assignee.id()
		}
		issues = append(issues, issue)
	}
	return issues, parsed.NextPageToken, parsed.IsLast, nil
}

// GetComments returns every comment on an issue, oldest first, following
// startAt pagination. Bodies are converted from ADF to Markdown.
func (c *Client) GetComments(key string) ([]Comment, error) {
	const pageSize = 100
	var comments []Comment
	for startAt := 0; ; {
		params := url.Values{
			"startAt":    {fmt.Sprint(startAt)},
			"maxResults": {fmt.Sprint(pageSize)},
		}
		u := c.baseURL + "/rest/api/3/issue/" + url.PathEscape(key) + "/comment?" + params.Encode()
		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		req.SetBasicAuth(c.email, c.apiToken)
		req.Header.Set("Accept", "application/json")

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		var parsed commentsResponse
		if resp.StatusCode >= 300 {
			resp.Body.Close()
			return nil, fmt.Errorf("jira: comments returned status %d", resp.StatusCode)
		}
		err = json.NewDecoder(resp.Body).Decode(&parsed)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		for _, i := range parsed.Comments {
			authorName := i.Author.DisplayName
			if authorName == "" {
				authorName = i.Author.id()
			}
			comments = append(comments, Comment{
				ID: string(i.ID), AuthorID: i.Author.id(), AuthorName: authorName,
				Body: rawToMarkdown(i.Body), ParentID: string(i.ParentID),
			})
		}
		startAt += len(parsed.Comments)
		if len(parsed.Comments) == 0 || (parsed.Total > 0 && startAt >= parsed.Total) {
			return comments, nil
		}
	}
}

// GetIssue fetches a single issue by key, used to validate a manual link (phase 10).
func (c *Client) GetIssue(key string) (*Issue, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/rest/api/3/issue/"+url.PathEscape(key)+"?fields=summary,status,issuetype", nil)
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
			Summary   string `json:"summary"`
			IssueType struct {
				Name string `json:"name"`
			} `json:"issuetype"`
			Status struct {
				Name string `json:"name"`
			} `json:"status"`
		} `json:"fields"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	return &Issue{Key: parsed.Key, Summary: parsed.Fields.Summary, Status: parsed.Fields.Status.Name, IssueType: parsed.Fields.IssueType.Name}, nil
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

// CreateIssue creates a new issue of the given type in the project and returns
// its key. A subtask passes its parent issue through parentKey. A task that
// belongs to an epic passes the epic key and the field Jira uses for that
// hierarchy in epicLinkField, which is "parent" on team-managed projects and a
// custom field such as customfield_10014 on company-managed ones.
func (c *Client) CreateIssue(projectKey, summary, issueType, parentKey, epicKey, epicLinkField string) (string, error) {
	fields := map[string]any{
		"project":   map[string]string{"key": projectKey},
		"summary":   summary,
		"issuetype": map[string]string{"name": issueType},
	}
	if parentKey != "" {
		fields["parent"] = map[string]string{"key": parentKey}
	}
	if epicKey != "" && epicLinkField != "" {
		if epicLinkField == "parent" {
			fields["parent"] = map[string]string{"key": epicKey}
		} else {
			fields[epicLinkField] = epicKey
		}
	}
	body, err := json.Marshal(map[string]any{"fields": fields})
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
