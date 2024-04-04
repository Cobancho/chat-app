package middleware

import (
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"strings"
)

func AuthMiddleware(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authorizationHeader := r.Header.Get("Authorization")

		if authorizationHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authorizationHeader, " ", 2)

		if len(parts) != 2 {
			log.Println("Invalid Authorization header")
			return
		}

		// Check if the authentication type is Basic
		if strings.ToLower(parts[0]) != "basic" {
			log.Println("Unsupported authentication type")
			return
		}

		// Decode the Base64-encoded credentials
		decoded, err := base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			log.Println("Error decoding credentials:", err)
			return
		}

		// Convert the decoded bytes to a string
		credentials := strings.ReplaceAll(string(decoded), ":", "")

		if credentials != os.Getenv("AUTH_TOKEN") {
			http.Error(w, "Authorization ID not valid", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	}
}
