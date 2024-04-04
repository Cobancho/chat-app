package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sistema-maika-chat/chats"
	"sistema-maika-chat/clients"
	"sistema-maika-chat/structs"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		ReadBufferSize:  512,
		WriteBufferSize: 512,
	}
)

func handleChatMessages(chatId int) {
	chat := chats.Chats[chatId]

	// Cuando se termina el go routine
	defer func() {
		log.Printf("Closing chat: %d\n", chatId)

		listKey := fmt.Sprintf("chat:%d", chatId)

		val, err := clients.Redis.ZRangeByScore(context.TODO(), listKey, &redis.ZRangeBy{
			Min: "-inf",
			Max: "+inf",
		}).Result()

		switch {
		case err == redis.Nil:
			log.Println("Key does not exist")
			return
		case err != nil:
			log.Println("Get failed", err)
			return
		case len(val) == 0:
			log.Println("No messages in redis")
			return
		}

		var messages []structs.Message
		for _, jsonStr := range val {
			var msg structs.Message
			err := json.Unmarshal([]byte(jsonStr), &msg)
			if err != nil {
				log.Fatal("Error unmarshalling JSON", err)
			}
			messages = append(messages, msg)
		}
		var insertionSlice []structs.InsertionMessage
		for _, msg := range messages {
			insertionMsg := structs.InsertionMessage{
				ChatId:    chatId,
				UserId:    msg.UserId,
				Message:   msg.Message,
				Timestamp: msg.Timestamp,
			}
			insertionSlice = append(insertionSlice, insertionMsg)
		}

		_, err = clients.DB.NamedExec(`INSERT INTO "Messages" ("chatId", "userId", timestamp, message)
      VALUES (:chat_id, :user_id, :timestamp, :message)`, insertionSlice)
		if err != nil {
			log.Println("Could not insert the messages into the database", err)
			return
		}
		log.Printf("Offloaded messages to database: %v", insertionSlice)
		res, err := clients.Redis.Del(context.TODO(), listKey).Result()

		if err != nil {
			log.Printf("Could not delete %s", listKey)
		}
		log.Printf("Redis key %s deleted correctly (%d)", listKey, res)
	}()

	for {
		msg, ok := <-chat.Broadcast
		if !ok {
			return
		}
		for client := range chat.Clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Println(err)
				client.Close()
				delete(chat.Clients, client)
			}
		}

		msgBytes, err := json.Marshal(msg)

		if err != nil {
			log.Println("Error serializing message to JSON", err)
		}

		listKey := fmt.Sprintf("chat:%d", chatId)

		err = clients.Redis.ZAdd(context.TODO(), listKey, redis.Z{Score: float64(msg.Timestamp), Member: msgBytes}).Err()
		if err != nil {
			log.Fatal("Could not push to Redis sorted set", err)
		}
	}
}

func createChat(chatId int) {
	chats.Chats[chatId] = &structs.Chat{
		Broadcast: make(chan structs.Message, 4),
		Clients:   make(map[*websocket.Conn]bool),
	}

	go handleChatMessages(chatId)
}

func HandleWebsocketConnection(w http.ResponseWriter, r *http.Request) {
	// Consigo el chatid del request
	chatIdString := r.URL.Query().Get("chatId")
	chatId, err := strconv.Atoi(chatIdString)
	if err != nil {
		http.Error(w, "chatId is not an integer", http.StatusBadRequest)
		return
	}

	if chatId == 0 {
		http.Error(w, "chatId cannot be 0", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Println(err)
		return
	}

	// Creo el chat si no existe
	if _, ok := chats.Chats[chatId]; !ok {
		createChat(chatId)
		log.Printf("Chat %d created", chatId)
	}

	chat := chats.Chats[chatId]

	// Asocio al cliente con el chat
	chat.Clients[conn] = true

	log.Println(fmt.Sprintf("Client connected on chat %d", chatId), chat.Clients)

	// Cuando se desconecta el cliente, chequeo si ya no quedan mas
	defer func() {
		_ = conn.Close()
		delete(chat.Clients, conn)
		log.Printf("Disconection from chat %d", chatId)
		if len(chat.Clients) == 0 {
			close(chat.Broadcast)
			delete(chats.Chats, chatId)
			log.Printf("Chat %d channel closed", chatId)
		}
	}()

	for {
		var msg structs.Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println(err)
			delete(chats.Chats[chatId].Clients, conn)
			break
		}

		log.Printf("Message recieved in chat %d: %v", chatId, msg)

		switch {
		case msg.UserId == "":
			conn.WriteMessage(websocket.TextMessage, []byte("UserId missing in body"))
			continue
		case msg.Message == "":
			conn.WriteMessage(websocket.TextMessage, []byte("Message missing in body"))
			continue
		case msg.Timestamp == 0:
			conn.WriteMessage(websocket.TextMessage, []byte("Timestamp missing in body"))
			continue
		}
		log.Printf("New message received from chat %d: %v\n", chatId, msg)
		chat.Broadcast <- msg
	}
}
