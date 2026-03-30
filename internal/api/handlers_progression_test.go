package api

import (
	"net/http/httptest"
	"testing"

	"github.com/h4ribote/Paper-Street/internal/auth"
	"github.com/h4ribote/Paper-Street/internal/engine"
	"github.com/h4ribote/Paper-Street/internal/models"
)

func TestDailyMissionCompletionAwardsReward(t *testing.T) {
	store := NewMarketStore()
	apiKeys := auth.NewAPIKeyCache()
	if err := apiKeys.AddHex(testAPIKeyUser1); err != nil {
		t.Fatalf("failed to add api key: %v", err)
	}
	store.RegisterAPIKey(testAPIKeyUser1, 1)
	store.EnsureUser(1)

	eng := engine.NewEngine(store)
	server := httptest.NewServer(NewRouter(eng, apiKeys, store))
	defer server.Close()

	var response DailyMissionResponse
	getJSON(t, server.URL+"/missions/daily?user_id=1", testAPIKeyUser1, &response)
	if len(response.Missions) < 2 {
		t.Fatalf("expected missions, got %+v", response.Missions)
	}
	var firstID, secondID string
	for _, mission := range response.Missions {
		if mission.Mission.Grade == "C" {
			if firstID == "" {
				firstID = mission.Mission.ID
			} else if secondID == "" {
				secondID = mission.Mission.ID
			}
		}
	}
	if firstID == "" || secondID == "" {
		t.Fatalf("expected grade C missions, got %+v", response.Missions)
	}

	var firstComplete MissionCompletionResult
	postJSON(t, server.URL+"/missions/"+firstID+"/complete", testAPIKeyUser1, missionCompleteRequest{UserID: 1}, &firstComplete)
	if firstComplete.Reward != nil {
		t.Fatalf("expected no reward on first completion, got %+v", firstComplete.Reward)
	}

	var secondComplete MissionCompletionResult
	postJSON(t, server.URL+"/missions/"+secondID+"/complete", testAPIKeyUser1, missionCompleteRequest{UserID: 1}, &secondComplete)
	if secondComplete.Reward == nil {
		t.Fatalf("expected reward on pair completion")
	}
	if secondComplete.Reward.XP != 5 || secondComplete.Reward.Cash != 400 {
		t.Fatalf("unexpected reward %+v", secondComplete.Reward)
	}
	if secondComplete.UserProgress.XP != 5 {
		t.Fatalf("expected xp 5, got %d", secondComplete.UserProgress.XP)
	}

	var balances []models.Balance
	getJSON(t, server.URL+"/portfolio/balances?user_id=1", testAPIKeyUser1, &balances)
	cash := balanceAmount(balances, defaultCurrency)
	if cash != defaultCashBalance+400 {
		t.Fatalf("expected cash %d, got %d", defaultCashBalance+400, cash)
	}
}

func TestContractDeliveryAwardsXP(t *testing.T) {
	store := NewMarketStore()
	apiKeys := auth.NewAPIKeyCache()
	if err := apiKeys.AddHex(testAPIKeyUser1); err != nil {
		t.Fatalf("failed to add api key: %v", err)
	}
	store.RegisterAPIKey(testAPIKeyUser1, 1)
	store.EnsureUser(1)

	store.mu.Lock()
	store.positions[1][101] = 25
	store.mu.Unlock()

	eng := engine.NewEngine(store)
	server := httptest.NewServer(NewRouter(eng, apiKeys, store))
	defer server.Close()

	var delivery ContractDeliveryResult
	postJSON(t, server.URL+"/contracts/2/deliver", testAPIKeyUser1, contractDeliveryRequest{UserID: 1, Quantity: 10}, &delivery)

	if delivery.Quantity != 10 {
		t.Fatalf("expected quantity 10, got %d", delivery.Quantity)
	}
	if delivery.CashReward != 1200 {
		t.Fatalf("expected cash reward 1200, got %d", delivery.CashReward)
	}
	if delivery.XPReward != 20 {
		t.Fatalf("expected xp reward 20, got %d", delivery.XPReward)
	}
	if delivery.Contract.UserDelivered != 10 {
		t.Fatalf("expected user delivered 10, got %d", delivery.Contract.UserDelivered)
	}
	if delivery.UserProgress.XP != 20 {
		t.Fatalf("expected xp 20, got %d", delivery.UserProgress.XP)
	}
}
