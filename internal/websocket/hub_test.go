package websocket

import "testing"

func TestClientSendAndMessagesCopy(t *testing.T) {
	client := NewClient("client-1")
	priceMessage := Message{Topic: "price", Data: map[string]int64{"value": 100}, TS: 1}
	newsMessage := Message{Topic: "news", Data: "headline", TS: 2}

	client.Send(priceMessage)
	client.Send(newsMessage)

	messages := client.Messages()
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	if messages[0].Topic != priceMessage.Topic || messages[1].Topic != newsMessage.Topic {
		t.Fatalf("unexpected message topics: %+v", messages)
	}

	messages[0].Topic = "mutated"
	latest := client.Messages()
	if latest[0].Topic != priceMessage.Topic {
		t.Fatalf("expected internal messages to stay an immutable copy, got %s", latest[0].Topic)
	}
}

func TestHubRegisterBroadcastAndUnregister(t *testing.T) {
	hub := NewHub()
	clientA := NewClient("a")
	clientB := NewClient("b")

	hub.Register(clientA)
	hub.Register(clientB)

	firstTickMessage := Message{Topic: "tick", Data: 101, TS: 10}
	hub.Broadcast(firstTickMessage)

	if got := clientA.Messages(); len(got) != 1 || got[0].Topic != firstTickMessage.Topic {
		t.Fatalf("clientA should receive first message, got %+v", got)
	}
	if got := clientB.Messages(); len(got) != 1 || got[0].Topic != firstTickMessage.Topic {
		t.Fatalf("clientB should receive first message, got %+v", got)
	}

	hub.Unregister(clientB)
	secondTickMessage := Message{Topic: "tick", Data: 102, TS: 11}
	hub.Broadcast(secondTickMessage)

	if got := clientA.Messages(); len(got) != 2 || got[1].Topic != secondTickMessage.Topic {
		t.Fatalf("clientA should receive second message, got %+v", got)
	}
	if got := clientB.Messages(); len(got) != 1 {
		t.Fatalf("clientB should not receive after unregister, got %+v", got)
	}
}
