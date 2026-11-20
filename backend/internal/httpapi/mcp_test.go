package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/alialjaffer/fai/backend/internal/auth"
	"github.com/alialjaffer/fai/backend/internal/store"
)

const testAgentSecret = "test-agent-secret-0123456789"

type agentTestServer struct {
	store  store.Store
	server *httptest.Server
}

func newAgentTestServer(t *testing.T) agentTestServer {
	t.Helper()
	st := testStore(t)
	handlers, err := auth.ParseAgentTokens("hermes:" + testAgentSecret)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(NewRouter(&Server{Store: st, AgentTokens: handlers}))
	t.Cleanup(server.Close)
	return agentTestServer{store: st, server: server}
}

func (ts agentTestServer) client(t *testing.T, secret string) *mcp.ClientSession {
	t.Helper()
	httpClient := &http.Client{Transport: bearerTransport{secret: secret, base: http.DefaultTransport}}
	client := mcp.NewClient(&mcp.Implementation{Name: "fai-test", Version: "1"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:             ts.server.URL + "/mcp",
		HTTPClient:           httpClient,
		DisableStandaloneSSE: true,
	}, nil)
	if err != nil {
		t.Fatalf("connecting MCP client: %v", err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

type bearerTransport struct {
	secret string
	base   http.RoundTripper
}

func (b bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "Bearer "+b.secret)
	return b.base.RoundTrip(clone)
}

func callTool(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return result
}

func decodeStructured[T any](t *testing.T, result *mcp.CallToolResult) T {
	t.Helper()
	if result.IsError {
		t.Fatalf("tool returned an error: %s", textContent(result))
	}
	payload, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var out T
	if err := json.Unmarshal(payload, &out); err != nil {
		t.Fatalf("decoding %s: %v", payload, err)
	}
	return out
}

func textContent(result *mcp.CallToolResult) string {
	var out strings.Builder
	for _, content := range result.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			out.WriteString(text.Text)
		}
	}
	return out.String()
}

func TestMCPRequiresAnAgentKey(t *testing.T) {
	ts := newAgentTestServer(t)
	res, err := http.Get(ts.server.URL + "/mcp")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated /mcp status = %d, want 401", res.StatusCode)
	}
}

func TestMCPTaskLifecycle(t *testing.T) {
	ts := newAgentTestServer(t)
	session := ts.client(t, testAgentSecret)

	contextOut := decodeStructured[getContextOut](t, callTool(t, session, "get_context", map[string]any{}))
	if contextOut.Today != time.Now().Format(mcpDateLayout) {
		t.Fatalf("today = %q", contextOut.Today)
	}
	if len(contextOut.Workspaces) == 0 || len(contextOut.Boards) == 0 {
		t.Fatalf("seeded context has no workspaces or boards: %+v", contextOut)
	}
	board := contextOut.Boards[0]
	if len(board.Columns) == 0 {
		t.Fatal("seeded board has no columns")
	}

	created := decodeStructured[mcpTask](t, callTool(t, session, "create_task", map[string]any{
		"title":    "Wire the agent surface",
		"board":    board.Name,
		"column":   board.Columns[0].Name,
		"due_date": "2026-02-01",
		"steps":    []string{"design", "implement"},
	}))
	if created.Ref == "" || created.Title != "Wire the agent surface" {
		t.Fatalf("created task = %+v", created)
	}

	detail := decodeStructured[taskDetail](t, callTool(t, session, "get_task", map[string]any{"task": created.Ref}))
	if len(detail.Subtasks) != 2 {
		t.Fatalf("subtasks = %d, want 2", len(detail.Subtasks))
	}

	list := decodeStructured[taskListOut](t, callTool(t, session, "list_tasks", map[string]any{"board": board.Name}))
	if list.Count == 0 || list.Tasks[0].Ref != created.Ref {
		t.Fatalf("list_tasks = %+v", list)
	}

	done := board.Columns[len(board.Columns)-1]
	if !done.IsDone {
		t.Skip("the seeded board's last column is not the done column")
	}
	completed := decodeStructured[mcpTask](t, callTool(t, session, "complete_task", map[string]any{"task": created.Ref}))
	if !completed.Completed || completed.Column != done.Name {
		t.Fatalf("completed = %+v", completed)
	}

	callTool(t, session, "add_comment", map[string]any{"task": created.Ref, "body": "done by the agent"})
	detail = decodeStructured[taskDetail](t, callTool(t, session, "get_task", map[string]any{"task": created.Ref}))
	if len(detail.Comments) != 1 || !strings.Contains(detail.Comments[0].Body, "done by the agent") {
		t.Fatalf("comments = %+v", detail.Comments)
	}
	if detail.Comments[0].Author != "agent:hermes" {
		t.Fatalf("comment author = %q", detail.Comments[0].Author)
	}
}

func TestMCPNotesAndUpcoming(t *testing.T) {
	ts := newAgentTestServer(t)
	session := ts.client(t, testAgentSecret)

	note := decodeStructured[noteOut](t, callTool(t, session, "create_note", map[string]any{
		"title":   "Runbook",
		"body_md": "Step one",
	}))
	if note.Note.Title != "Runbook" {
		t.Fatalf("note = %+v", note)
	}
	appended := decodeStructured[noteOut](t, callTool(t, session, "append_note", map[string]any{
		"note": "Runbook", "body_md": "Step two",
	}))
	if !strings.Contains(appended.Note.BodyMd, "Step one") || !strings.Contains(appended.Note.BodyMd, "Step two") {
		t.Fatalf("appended body = %q", appended.Note.BodyMd)
	}

	created := decodeStructured[mcpTask](t, callTool(t, session, "create_task", map[string]any{
		"title": "Ship v1", "due_date": "2020-01-01",
	}))
	upcoming := decodeStructured[upcomingOut](t, callTool(t, session, "list_upcoming", map[string]any{"days": 30}))
	found := false
	for _, task := range upcoming.Tasks {
		if task.Ref == created.Ref {
			found = true
		}
	}
	if !found {
		t.Fatalf("overdue task %s missing from upcoming: %+v", created.Ref, upcoming.Tasks)
	}
}

func TestMCPKeyScopedToWorkspace(t *testing.T) {
	ts := newAgentTestServer(t)
	ctx := context.Background()
	contextOut := decodeStructured[getContextOut](t, callTool(t, ts.client(t, testAgentSecret), "get_context", map[string]any{}))
	if len(contextOut.Workspaces) == 0 {
		t.Fatal("no workspaces seeded")
	}
	home := contextOut.Workspaces[0]
	other, err := ts.store.CreateWorkspace(ctx, "Other", "OTHR")
	if err != nil {
		t.Fatal(err)
	}
	secret := "fai_scoped-agent-secret-0123456789"
	if _, err := ts.store.CreateAgentToken(ctx, "scoped", hashAgentSecret(secret), "fai_scoped…", &home.ID); err != nil {
		t.Fatal(err)
	}
	session := ts.client(t, secret)

	// A board in the key's own workspace is allowed.
	allowed := decodeStructured[mcpBoard](t, callTool(t, session, "create_board", map[string]any{
		"name": "Scoped board", "workspace": home.Tag,
	}))
	if allowed.Workspace != home.Tag {
		t.Fatalf("created board = %+v", allowed)
	}

	// A board in another workspace is refused.
	refused := callTool(t, session, "create_board", map[string]any{
		"name": "Should fail", "workspace": other.Tag,
	})
	if !refused.IsError {
		t.Fatalf("scoped key created a board in %s", other.Tag)
	}
}

func TestMCPScopedKeyCannotListEveryWorkspace(t *testing.T) {
	ts := newAgentTestServer(t)
	ctx := context.Background()
	unscoped := ts.client(t, testAgentSecret)
	created := decodeStructured[mcpTask](t, callTool(t, unscoped, "create_task", map[string]any{"title": "Visible only in the home workspace"}))

	empty, err := ts.store.CreateWorkspace(ctx, "Empty", "EMPT")
	if err != nil {
		t.Fatal(err)
	}
	secret := "fai_empty-workspace-key-0123456789"
	if _, err := ts.store.CreateAgentToken(ctx, "empty", hashAgentSecret(secret), "fai_empty…", &empty.ID); err != nil {
		t.Fatal(err)
	}
	listed := decodeStructured[taskListOut](t, callTool(t, ts.client(t, secret), "list_tasks", map[string]any{}))
	if listed.Count != 0 {
		t.Fatalf("scoped key in an empty workspace saw %d tasks (first %+v, wanted none including %s)", listed.Count, listed.Tasks[0], created.Ref)
	}
}

func TestAgentTokenEndpoints(t *testing.T) {
	ts := newAgentTestServer(t)

	res := httpGet(t, ts.server.URL+"/api/agent-tokens")
	listed := decodeInto[[]store.AgentToken](t, res)
	if len(listed) != 0 {
		t.Fatalf("new instance has %d agent keys", len(listed))
	}

	res = postJSON(t, ts.server.URL+"/api/agent-tokens", `{"name":"Claude Code"}`)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d", res.StatusCode)
	}
	created := decodeInto[createdAgentToken](t, res)
	if !strings.HasPrefix(created.Secret, agentSecretPrefix) || created.Token.Prefix == "" {
		t.Fatalf("created key = %+v", created)
	}
	if strings.Contains(created.Token.Prefix, created.Secret) {
		t.Fatal("the stored prefix leaks the secret")
	}

	// The key works over MCP.
	clone := ts.client(t, created.Secret)
	if out := decodeStructured[getContextOut](t, callTool(t, clone, "get_context", map[string]any{})); out.Agent != "Claude Code" {
		t.Fatalf("agent name = %q", out.Agent)
	}

	// last_used_at is written in the background at most once a minute.
	deadline := time.Now().Add(3 * time.Second)
	for {
		res = httpGet(t, ts.server.URL+"/api/agent-tokens")
		listed = decodeInto[[]store.AgentToken](t, res)
		if len(listed) == 1 && listed[0].LastUsedAt != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("list after create = %+v", listed)
		}
		time.Sleep(50 * time.Millisecond)
	}

	req, err := http.NewRequest(http.MethodDelete, ts.server.URL+"/api/agent-tokens/1", nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d", res.StatusCode)
	}

	// A revoked key no longer authenticates.
	httpClient := &http.Client{Transport: bearerTransport{secret: created.Secret, base: http.DefaultTransport}}
	client := mcp.NewClient(&mcp.Implementation{Name: "fai-test", Version: "1"}, nil)
	if _, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint: ts.server.URL + "/mcp", HTTPClient: httpClient, DisableStandaloneSSE: true,
	}, nil); err == nil {
		t.Fatal("a revoked key still connects")
	}
}
