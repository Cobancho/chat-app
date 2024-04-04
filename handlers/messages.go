package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sistema-maika-chat/chats"
	"sistema-maika-chat/clients"
	"sistema-maika-chat/structs"
	"strconv"
)

func HandlePostMessage(w http.ResponseWriter, r *http.Request) {
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

	var msg structs.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	log.Printf("New message received for chat %d: %v\n", chatId, msg)
	w.Write([]byte("Message succesfully received by the server"))

	if _, ok := chats.Chats[chatId]; !ok {
		insertionMessage := structs.InsertionMessage{
			ChatId:    chatId,
			Timestamp: msg.Timestamp,
			UserId:    msg.UserId,
			Message:   msg.Message,
		}

		_, err = clients.DB.NamedExec(`INSERT INTO "Messages" ("chatId", "userId", timestamp, message)
      VALUES (:chat_id, :user_id, :timestamp, :message)`, insertionMessage)
		if err != nil {
			log.Println("Could not insert the message into the database", err)
			return
		}
		log.Printf("Offloaded message to database directly: %v", insertionMessage)
		return
	}

	chats.Chats[chatId].Broadcast <- msg

}

func HandleDeleteMessage(w http.ResponseWriter, r *http.Request) {
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

	if _, ok := chats.Chats[chatId]; !ok {
		http.Error(w, "chatId does not exists", http.StatusBadRequest)
		return
	}

	var msg structs.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	log.Printf("Delete message request received from chat %d: %v\n", chatId, msg)

	listKey := fmt.Sprintf("chat:%d", chatId)

	err = clients.Redis.ZRemRangeByScore(r.Context(), listKey, fmt.Sprintf("%d", msg.Timestamp), fmt.Sprintf("%d", msg.Timestamp)).Err()

	if err != nil {
		log.Fatal("Could not delete message from Redis sorted set", err)
	}

	log.Printf("Message deleted from redis")

	w.Write([]byte("Message succesfully received by the server"))
}
