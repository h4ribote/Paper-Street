package api

import (
	"testing"

	"github.com/h4ribote/Paper-Street/internal/models"
)

func TestShouldSeedInitialAllocations(t *testing.T) {
	tests := []struct {
		name                 string
		users                []models.User
		roleToAPIKeyCount    int
		currencyBalanceCount int
		assetBalanceCount    int
		want                 bool
	}{
		{
			name:  "empty users triggers seed",
			users: nil,
			want:  true,
		},
		{
			name: "first db boot with only system user triggers seed",
			users: []models.User{
				{ID: 1, Username: "Paper Street Insurance Fund"},
			},
			want: true,
		},
		{
			name: "existing api keys skip seed",
			users: []models.User{
				{ID: 1, Username: "Paper Street Insurance Fund"},
			},
			roleToAPIKeyCount: 1,
			want:              false,
		},
		{
			name: "existing balances skip seed",
			users: []models.User{
				{ID: 1, Username: "Paper Street Insurance Fund"},
			},
			currencyBalanceCount: 1,
			want:                 false,
		},
		{
			name: "multiple users skip seed",
			users: []models.User{
				{ID: 1, Username: "Paper Street Insurance Fund"},
				{ID: 1000, Username: "Market Maker"},
			},
			want: false,
		},
		{
			name: "single non-system user skips seed",
			users: []models.User{
				{ID: 999, Username: "existing"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &MarketStore{
				roleToAPIKey: make(map[string]string),
			}
			for i := 0; i < tt.roleToAPIKeyCount; i++ {
				store.roleToAPIKey[string(rune('a'+i))] = "k"
			}
			got := store.shouldSeedInitialAllocations(tt.users, tt.currencyBalanceCount, tt.assetBalanceCount)
			if got != tt.want {
				t.Fatalf("shouldSeedInitialAllocations() = %v, want %v", got, tt.want)
			}
		})
	}
}
