package db

type Queries struct {
	Conn *Connection
}

func NewQueries(conn *Connection) *Queries {
	return &Queries{Conn: conn}
}
