package dnsdelivery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

func TestDeleteSendsRecordSelectors(t *testing.T) {
	t.Helper()

	var deleteQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/dns/domain/get":
			if r.URL.Query().Get("domain") != "media.example.com" {
				t.Errorf("domain = %q", r.URL.Query().Get("domain"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "data": map[string]string{"zone_id": "zone-1"}})
		case "/v1/dns/record/delete":
			if r.Method != http.MethodDelete {
				t.Errorf("method = %s", r.Method)
			}
			deleteQuery = r.URL.Query()
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "data": nil})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient("test-key")
	client.baseURL = server.URL
	if err := client.Delete(context.Background(), "media.example.com", "record-1", "CNAME", "clip.media.example.com"); err != nil {
		t.Fatal(err)
	}
	want := url.Values{
		"zone_id":     {"zone-1"},
		"record_id":   {"record-1"},
		"record_type": {"CNAME"},
		"name":        {"clip.media.example.com"},
	}
	if !reflect.DeepEqual(deleteQuery, want) {
		t.Fatalf("delete query = %v, want %v", deleteQuery, want)
	}
}

func TestDeleteRejectsUnscopedRequest(t *testing.T) {
	client := NewClient("test-key")
	if err := client.Delete(context.Background(), "media.example.com", "", "", ""); err == nil {
		t.Fatal("Delete returned nil error")
	}
}
