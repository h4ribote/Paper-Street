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
	store.SetPosition(1, target.AssetID, 25)

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
	store.mu.Lock()
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

func TestXPBarrierUpgradesRank(t *testing.T) {
	store := NewMarketStore()
	store.EnsureUser(1)

	// Base XP is 0. Shrimp rank.
	info, _ := store.UserRankInfo(1)
	if info.Rank != "Shrimp" {
		t.Fatalf("expected Rank Shrimp at 0 XP, got %s", info.Rank)
	}

	// 499 XP is not enough for Fish (requires 500 XP)
	store.AddXP(1, 499)
	info, _ = store.UserRankInfo(1)
	if info.Rank != "Shrimp" {
		t.Fatalf("expected Rank Shrimp at 499 XP, got %s", info.Rank)
	}

	// Exactly 500 XP crosses the barrier to Fish
	store.AddXP(1, 1)
	info, _ = store.UserRankInfo(1)
	if info.Rank != "Fish" {
		t.Fatalf("expected Rank Fish at 500 XP, got %s", info.Rank)
	}
}

func TestRankUpgradeLowersFees(t *testing.T) {
	store := NewMarketStore()
	apiKeys := auth.NewAPIKeyCache()
	if err := apiKeys.AddHex(testAPIKeyUser1); err != nil {
		t.Fatalf("failed to add api key: %v", err)
	}
	if err := apiKeys.AddHex(testAPIKeyUser2); err != nil {
		t.Fatalf("failed to add api key: %v", err)
	}
	store.RegisterAPIKey(testAPIKeyUser1, 1)
	store.RegisterAPIKey(testAPIKeyUser2, 2)
	store.EnsureUser(1)
	store.EnsureUser(2)
	store.SetBalance(1, defaultCurrency, 100_000_000)
	store.SetBalance(2, defaultCurrency, 100_000_000)
	store.SetPosition(2, 101, 10000)
	store.SetPosition(2, 102, 10000)

	eng := engine.NewEngine(nil, store)
	server := httptest.NewServer(NewRouter(eng, apiKeys, store, ""))
	defer server.Close()

	// 1. Initial order as Shrimp
	postJSON(t, server.URL+"/api/orders", testAPIKeyUser2, map[string]interface{}{
		"asset_id": 101, "user_id": 2, "side": "SELL", "type": "LIMIT", "quantity": 10000, "price": 100,
	}, nil)

	postJSON(t, server.URL+"/api/orders", testAPIKeyUser1, map[string]interface{}{
		"asset_id": 101, "user_id": 1, "side": "BUY", "type": "MARKET", "quantity": 10000,
	}, nil)

	time.Sleep(100 * time.Millisecond)

	cash1 := store.GetBalance(1, defaultCurrency)
	fee1 := 100_000_000 - cash1 - 1000000 // 1,000,000 is base cost (10000 * 100)

	if fee1 <= 0 {
		t.Fatalf("expected positive fee to be charged, got %d", fee1)
	}

	// 2. Upgrade Rank to Whale (requires 15,000 XP)
	store.AddXP(1, 15_000)

	cashBefore2 := store.GetBalance(1, defaultCurrency)

	postJSON(t, server.URL+"/api/orders", testAPIKeyUser2, map[string]interface{}{
		"asset_id": 102, "user_id": 2, "side": "SELL", "type": "LIMIT", "quantity": 10000, "price": 100,
	}, nil)

	postJSON(t, server.URL+"/api/orders", testAPIKeyUser1, map[string]interface{}{
		"asset_id": 102, "user_id": 1, "side": "BUY", "type": "MARKET", "quantity": 10000,
	}, nil)

	time.Sleep(100 * time.Millisecond)

	cashAfter := store.GetBalance(1, defaultCurrency)
	fee2 := cashBefore2 - cashAfter - 1000000

	if fee2 >= fee1 {
		t.Fatalf("expected fee to actively lower after rank upgrade. Before: %d, After: %d", fee1, fee2)
	}
}
