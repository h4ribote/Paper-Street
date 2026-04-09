package api

import (
	"context"
	"log"
)

func (s *MarketStore) loadIndexesFromDB(ctx context.Context) error {
	if s.queries == nil {
		return nil
	}
	records, err := s.queries.ListIndexConstituents(ctx)
	if err != nil {
		return err
	}
	// Group components by index asset ID.
	constituentMap := make(map[int64][]int64)
	for _, r := range records {
		constituentMap[r.IndexAssetID] = append(constituentMap[r.IndexAssetID], r.ComponentAssetID)
	}
	for indexAssetID, components := range constituentMap {
		definition := IndexDefinition{
			AssetID:    indexAssetID,
			Components: components,
			FeeBps:     indexFeeBps,
		}
		s.indexes[indexAssetID] = definition
		
		asset := s.ensureAssetLocked(indexAssetID)
		asset.Type = "INDEX"
		if asset.Sector == "" {
			asset.Sector = "MIXED"
		}
		
		s.updateAssetLocked(asset, s.indexUnitPriceLocked(definition))
	}
	return nil
}

func (s *MarketStore) persistIndex(definition IndexDefinition) {
	if s.queries == nil {
		return
	}
	ctx, cancel := s.dbContext()
	defer cancel()
	if err := s.queries.UpsertIndexConstituents(ctx, definition.AssetID, definition.Components); err != nil {
		log.Printf("db upsert index constituents %d: %v", definition.AssetID, err)
	}
}
