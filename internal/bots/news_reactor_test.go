package bots

import "testing"

func TestReactionOrder(t *testing.T) {
	event := NewsEvent{AssetID: 1, Sentiment: 0.5, Confidence: 0.8}
	reaction, ok := ReactionOrder(event, 10, 0.1)
	if !ok {
		t.Fatal("expected reaction to be generated")
	}
	if reaction.Side != "BUY" {
		t.Fatalf("expected BUY reaction, got %s", reaction.Side)
	}
	if reaction.Quantity != 4 {
		t.Fatalf("expected quantity 4, got %d", reaction.Quantity)
	}
}

func TestReactionOrderNegative(t *testing.T) {
	event := NewsEvent{AssetID: 1, Sentiment: -0.9, Confidence: 1}
	reaction, ok := ReactionOrder(event, 10, 0.1)
	if !ok {
		t.Fatal("expected reaction to be generated")
	}
	if reaction.Side != "SELL" {
		t.Fatalf("expected SELL reaction, got %s", reaction.Side)
	}
	if reaction.Quantity != 9 {
		t.Fatalf("expected quantity 9, got %d", reaction.Quantity)
	}
}

func TestReactionOrderLowConfidence(t *testing.T) {
	event := NewsEvent{AssetID: 1, Sentiment: 0.9, Confidence: 0.05}
	_, ok := ReactionOrder(event, 10, 0.1)
	if ok {
		t.Fatal("expected no reaction for low confidence")
	}
}
