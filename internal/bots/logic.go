package bots

type Bot struct {
	ID       int64
	Strategy Strategy
}

func NewBot(id int64, strategy Strategy) *Bot {
	return &Bot{ID: id, Strategy: strategy}
}
