# Chat Application
Chat application written in Go.

Using [gorilla/websocket](https://github.com/gorilla/websocket) for broadcasting messages to connected users.
Incomming messages are handled by an API Endpoint, as well as deleting them.
The messages during a session are stored in a Redis sorted list and when all users disconnect and thus the session ends, the messages are offloaded to a PostgreSQL database for long-term storage.
