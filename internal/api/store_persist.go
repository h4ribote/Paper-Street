package api

import (
	"log"
	"strings"
	"time"

	"github.com/h4ribote/Paper-Street/internal/db"
	"github.com/h4ribote/Paper-Street/internal/models"
)

func (s *MarketStore) ensureCurrencyID(currency string) int64 {
	if s == nil || s.queries == nil {
		return 0
	}
	currency = strings.TrimSpace(currency)
	if currency == "" {
		return 0
	}
	if id, ok := s.currencyIDs[currency]; ok {
		return id
	}
	ctx, cancel := s.dbContext()
	defer cancel()
	id, err := s.queries.EnsureDefaultCurrency(ctx, currency)
	if err != nil {
		log.Printf("db ensure currency %s: %v", currency, err)
		return 0
	}
	s.currencyIDs[currency] = id
	return id
}

func (s *MarketStore) persistCurrencyBalance(userID int64, currency string, amount int64) {
	if s == nil || s.queries == nil || userID == 0 {
		return
	}
	currencyID := s.ensureCurrencyID(currency)
	if currencyID == 0 {
		return
	}
	ctx, cancel := s.dbContext()
	defer cancel()

	val, err := s.queries.GetBalance(ctx, models.GetBalanceParams{UserID: userID, Currency: currency})
	delta := amount - val
	if err == nil && delta != 0 {
		_ = s.queries.InsertTransactionLog(ctx, nil, db.TransactionLogRecord{
			UserID:       userID,
			CurrencyID:   currencyID,
			Amount:       delta,
			BalanceAfter: amount,
			Type:         "BALANCE_UPDATE",
			ReferenceID:  "SYNC",
			Description:  "Currency balance persisted",
			CreatedAt:    time.Now().UnixMilli(),
		})
	}

	if err := s.queries.SetCurrencyBalance(ctx, userID, currencyID, amount); err != nil {
		log.Printf("db set currency balance %d/%s: %v", userID, currency, err)
	}
}

func (s *MarketStore) persistAssetBalance(userID, assetID, quantity int64) {
	if s == nil || s.queries == nil || userID == 0 || assetID == 0 {
		return
	}
	ctx, cancel := s.dbContext()
	defer cancel()
	if err := s.queries.SetAssetBalance(ctx, userID, assetID, quantity); err != nil {
		log.Printf("db set asset balance %d/%d: %v", userID, assetID, err)
	}
}

func (s *MarketStore) persistAPIKey(role, key string, userID int64) {
	if s == nil || s.queries == nil || userID == 0 {
		return
	}
	role = strings.TrimSpace(role)
	key = strings.TrimSpace(key)
	if role == "" || key == "" {
		return
	}
	ctx, cancel := s.dbContext()
	defer cancel()
	record := db.APIKeyRecord{Key: key, UserID: userID, Role: role}
	if err := s.queries.UpsertAPIKey(ctx, record, time.Now().UTC()); err != nil {
		log.Printf("db upsert api key %s: %v", role, err)
	}
}
