package sdk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoginLegacySessionMintsPAT(t *testing.T) {
	var sawNewAPIUser string
	var sawCookie bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/user/login":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "legacy-sess"})
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"message": "",
				"data": map[string]any{
					"id":           7,
					"username":     "alice",
					"display_name": "Alice",
					"role":         1,
					"status":       1,
					"group":        "default",
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/user/token":
			sawNewAPIUser = r.Header.Get("New-Api-User")
			if c, err := r.Cookie("session"); err == nil && c.Value == "legacy-sess" {
				sawCookie = true
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"message": "",
				"data":    "legacy-pat-token",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	result, err := c.Login(context.Background(), "alice", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if result.AccessToken != "legacy-pat-token" {
		t.Fatalf("token=%q", result.AccessToken)
	}
	if result.User.ID != 7 || result.User.Username != "alice" {
		t.Fatalf("user=%+v", result.User)
	}
	if c.AccessToken != "legacy-pat-token" || c.UserID != 7 {
		t.Fatalf("client token=%q userID=%d", c.AccessToken, c.UserID)
	}
	if sawNewAPIUser != "7" {
		t.Fatalf("New-Api-User=%q", sawNewAPIUser)
	}
	if !sawCookie {
		t.Fatal("expected session cookie on /api/user/token")
	}
}

func TestLoginNewJWTUpgradesToPAT(t *testing.T) {
	var sawAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/user/login":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"access_token":      "jwt-token",
					"token_type":        "Bearer",
					"access_expires_at": 1,
					"user": map[string]any{
						"id":       9,
						"username": "bob",
					},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/user/token":
			sawAuth = r.Header.Get("Authorization")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"message": "",
				"data":    "long-lived-pat",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	result, err := c.Login(context.Background(), "bob", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if result.AccessToken != "long-lived-pat" || c.AccessToken != "long-lived-pat" || c.UserID != 9 {
		t.Fatalf("result=%+v clientToken=%q user=%d", result, c.AccessToken, c.UserID)
	}
	if sawAuth != "Bearer jwt-token" {
		t.Fatalf("Authorization=%q", sawAuth)
	}
	if result.AccessExpiresAt != 0 {
		t.Fatalf("AccessExpiresAt=%d want 0 after PAT upgrade", result.AccessExpiresAt)
	}
}

func TestLoginNewJWTKeepsJWTWhenPATMintFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/user/login":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"data": map[string]any{
					"access_token": "jwt-token",
					"user": map[string]any{
						"id":       9,
						"username": "bob",
					},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/user/token":
			http.Error(w, "forbidden", http.StatusForbidden)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	result, err := c.Login(context.Background(), "bob", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if result.AccessToken != "jwt-token" {
		t.Fatalf("token=%q", result.AccessToken)
	}
}
