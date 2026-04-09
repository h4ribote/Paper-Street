package api

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

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

	eng := engine.NewEngine(nil, store)
	server := httptest.NewServer(NewRouter(eng, apiKeys, store, ""))
	defer server.Close()

	var response DailyMissionResponse
	getJSON(t, server.URL+"/api/missions/daily?user_id=1", testAPIKeyUser1, &response)
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
	postJSON(t, server.URL+"/api/missions/"+firstID+"/complete", testAPIKeyUser1, missionCompleteRequest{UserID: 1}, &firstComplete)
	if firstComplete.Reward != nil {
		t.Fatalf("expected no reward on first completion, got %+v", firstComplete.Reward)
	}

	var secondComplete MissionCompletionResult
	postJSON(t, server.URL+"/api/missions/"+secondID+"/complete", testAPIKeyUser1, missionCompleteRequest{UserID: 1}, &secondComplete)
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
	getJSON(t, server.URL+"/api/portfolio/balances?user_id=1", testAPIKeyUser1, &balances)
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

	eng := engine.NewEngine(nil, store)
	server := httptest.NewServer(NewRouter(eng, apiKeys, store, ""))
	defer server.Close()

	var contracts []ContractStatus
	getJSON(t, server.URL+"/api/contracts?user_id=1", testAPIKeyUser1, &contracts)
	if len(contracts) == 0 {
		t.Fatalf("expected contracts, got %v", contracts)
	}
	var target ContractStatus
	userRank, _ := rankDefinitionByName(defaultRankName)
	for _, contract := range contracts {
		minRank, ok := rankDefinitionByName(contract.MinRank)
		if !ok || userRank.ID >= minRank.ID {
			target = contract
			break
		}
	}
	if target.ID == 0 {
		target = contracts[0]
	}

	var delivery ContractDeliveryResult
	store.mu.Lock()
	store.SetPosition(11, target.AssetID, 25)
	store.mu.Unlock()

	postJSON(t, server.URL+"/api/contracts/"+fmt.Sprint(target.ID)+"/deliver", testAPIKeyUser1, contractDeliveryRequest{UserID: 1, Quantity: 10}, &delivery)

	if delivery.Quantity != 10 {
		t.Fatalf("expected quantity 10, got %d", delivery.Quantity)
	}
	expectedCash, ok := safeMultiplyInt64(delivery.Quantity, target.PricePerUnit)
	if !ok {
		t.Fatalf("cash reward overflow")
	}
	if delivery.CashReward != expectedCash {
		t.Fatalf("expected cash reward %d, got %d", expectedCash, delivery.CashReward)
	}
	expectedXP, ok := safeMultiplyInt64(delivery.Quantity, target.XPPerUnit)
	if !ok {
		t.Fatalf("xp reward overflow")
	}
	if delivery.XPReward != expectedXP {
		t.Fatalf("expected xp reward %d, got %d", expectedXP, delivery.XPReward)
	}
	if delivery.Contract.UserDelivered != 10 {
		t.Fatalf("expected user delivered 10, got %d", delivery.Contract.UserDelivered)
	}
	if delivery.UserProgress.XP != expectedXP {
		t.Fatalf("expected xp %d, got %d", expectedXP, delivery.UserProgress.XP)
	}
}

func TestContractPriceUsesVWAPPremium(t *testing.T) {
	store := NewMarketStore()
	now := time.Now().UTC()
	store.mu.Lock()
	store.AddExecution(engine.Execution{
		AssetID:       contractAssetAUR,
		Price:         100,
		Quantity:      10,
		OccurredAtUTC: now.Add(-1 * time.Hour),
	})
	store.AddExecution(engine.Execution{
		AssetID:       contractAssetAUR,
		Price:         200,
		Quantity:      10,
		OccurredAtUTC: now.Add(-30 * time.Minute),
	})
	premium := store.contractPremiumBpsLocked(contractAssetAUR, contractKindProcurement)
	price := store.calculateContractPriceLocked(contractAssetAUR, contractKindProcurement, now)
	store.mu.Unlock()

	vwap := int64(150)
	scaled, ok := safeMultiplyInt64(vwap, 10_000+premium)
	if !ok {
		t.Fatalf("price overflow")
	}
	expected := roundUpDiv(scaled, 10_000)
	if price != expected {
		t.Fatalf("expected price %d, got %d", expected, price)
	}
}
