package sdk

import (
	"encoding/json"
	"testing"
)

func TestFlexibleCodeUnmarshal(t *testing.T) {
	cases := []struct {
		name   string
		raw    string
		truthy bool
	}{
		{"bool_true", `{"success":false,"message":"","code":true,"data":null}`, true},
		{"num_one", `{"success":false,"message":"","code":1,"data":null}`, true},
		{"num_401", `{"success":false,"message":"unauthorized","code":401,"data":null}`, false},
		{"str_auth", `{"success":false,"message":"x","code":"AUTH_UNAUTHORIZED"}`, false},
		{"str_true", `{"success":true,"message":"","code":"true"}`, true},
		{"omitted_ok_via_success", `{"success":true,"message":"","data":{}}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var env apiEnvelope
			if err := json.Unmarshal([]byte(tc.raw), &env); err != nil {
				t.Fatal(err)
			}
			if env.Code.Truthy != tc.truthy {
				t.Fatalf("Truthy=%v want %v", env.Code.Truthy, tc.truthy)
			}
		})
	}
}
