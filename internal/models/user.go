package models

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	RankID   int    `json:"rank_id"`
	Rank     string `json:"rank"`
	XP       int64  `json:"xp"`
}
