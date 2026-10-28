package httpapi

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/alialjaffer/fai/backend/internal/store"
)

func TestBoardAndColumnEditing(t *testing.T) {
	_, server := newWorkspaceServer(t)

	res := patchJSON(t, server.URL+"/api/boards/1", `{"name":"Renamed"}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("rename board status = %d", res.StatusCode)
	}
	if board := decodeInto[store.Board](t, res); board.Name != "Renamed" {
		t.Fatalf("board name = %q", board.Name)
	}

	res = httpGet(t, server.URL+"/api/boards/1/columns")
	columns := decodeInto[[]store.BoardColumn](t, res)
	if len(columns) < 2 {
		t.Fatalf("seed board has %d columns", len(columns))
	}

	first := columns[0]
	res = patchJSON(t, server.URL+"/api/boards/1/columns/"+strconv.FormatInt(first.ID, 10),
		`{"name":"Backlog","is_done":false}`)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("rename column status = %d", res.StatusCode)
	}
	if column := decodeInto[store.BoardColumn](t, res); column.Name != "Backlog" {
		t.Fatalf("column name = %q", column.Name)
	}

	ids := make([]int64, len(columns))
	for i, column := range columns {
		ids[len(columns)-1-i] = column.ID
	}
	body := `{"ids":[`
	for i, id := range ids {
		if i > 0 {
			body += ","
		}
		body += strconv.FormatInt(id, 10)
	}
	body += `]}`
	res = patchJSON(t, server.URL+"/api/boards/1/columns", body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("reorder status = %d", res.StatusCode)
	}
	reordered := decodeInto[[]store.BoardColumn](t, res)
	if reordered[0].ID != ids[0] || reordered[len(reordered)-1].ID != ids[len(ids)-1] {
		t.Fatalf("reordered ids = %#v, want %#v", reordered, ids)
	}

	res = patchJSON(t, server.URL+"/api/boards/1/columns", `{"ids":[1,2]}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("partial reorder status = %d", res.StatusCode)
	}
	res.Body.Close()
}
