package structs

import "github.com/gorilla/websocket"

type Message struct {
	UserId    string `json:"userid"`
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message"`
}

type InsertionMessage struct {
	ChatId    int    `db:"chat_id"`
	UserId    string `db:"user_id"`
	Timestamp int64  `db:"timestamp"`
	Message   string `db:"message"`
}

type Chat struct {
	Broadcast chan Message
	Clients   map[*websocket.Conn]bool
}
