package common

import (
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateNumericVerificationCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		length     int
		wantLength int
	}{
		{name: "default six digits", length: 6, wantLength: 6},
		{name: "four digits", length: 4, wantLength: 4},
		{name: "five digits", length: 5, wantLength: 5},
		{name: "clamp below four", length: 3, wantLength: 4},
		{name: "clamp above six", length: 8, wantLength: 6},
		{name: "clamp zero", length: 0, wantLength: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			code := GenerateNumericVerificationCode(tt.length)
			require.Len(t, code, tt.wantLength)
			for _, r := range code {
				assert.True(t, unicode.IsDigit(r), "code %q contains non-digit %q", code, r)
			}
		})
	}
}
