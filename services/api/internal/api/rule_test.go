package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tale-backend/internal/realtime"
)

func TestRuleCRUDPersistsChanges(t *testing.T) {
	rulesPath := filepath.Join(t.TempDir(), "rules.json")
	if err := os.WriteFile(rulesPath, []byte("[]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := realtime.LoadRules(rulesPath); err != nil {
		t.Fatal(err)
	}

	create := requestRule(t, http.MethodPost, "/rule/items", `{
      "id":"bath",
      "keywords":["温泉"],
      "question":"温泉はどこですか？",
      "answer":"2階です",
      "arActions":[]
    }`)
	if create.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", create.Code, create.Body.String())
	}

	list := httptest.NewRecorder()
	handleRuleCollection(list, httptest.NewRequest(http.MethodGet, "/rule/items", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d", list.Code)
	}
	var rules []realtime.Rule
	if err := json.Unmarshal(list.Body.Bytes(), &rules); err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].ID != "bath" {
		t.Fatalf("listed rules = %#v", rules)
	}

	update := requestRule(t, http.MethodPut, "/rule/items/bath", `{
      "keywords":["温泉","風呂"],
      "question":"お風呂はどこですか？",
      "answer":"2階にあります",
      "arActions":[]
    }`)
	if update.Code != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", update.Code, update.Body.String())
	}

	remove := requestRule(t, http.MethodDelete, "/rule/items/bath", "")
	if remove.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, body = %s", remove.Code, remove.Body.String())
	}

	data, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != "[]" {
		t.Fatalf("persisted rules = %s", data)
	}
}

func TestRuleCreateRejectsInvalidInput(t *testing.T) {
	rulesPath := filepath.Join(t.TempDir(), "rules.json")
	if err := os.WriteFile(rulesPath, []byte("[]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := realtime.LoadRules(rulesPath); err != nil {
		t.Fatal(err)
	}

	response := requestRule(t, http.MethodPost, "/rule/items", `{"question":"missing fields"}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestRuleCreateRejectsMultipleJSONValues(t *testing.T) {
	rulesPath := filepath.Join(t.TempDir(), "rules.json")
	if err := os.WriteFile(rulesPath, []byte("[]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := realtime.LoadRules(rulesPath); err != nil {
		t.Fatal(err)
	}

	response := requestRule(t, http.MethodPost, "/rule/items", `{"keywords":["one"],"question":"q","answer":"a"} {}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func requestRule(t *testing.T, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	response := httptest.NewRecorder()
	if target == "/rule/items" {
		handleRuleCollection(response, request)
	} else {
		handleRuleItem(response, request)
	}
	return response
}
