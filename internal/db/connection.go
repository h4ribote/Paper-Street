package db

type Connection struct {
	DSN string
}

func NewConnection(dsn string) *Connection {
	return &Connection{DSN: dsn}
}
