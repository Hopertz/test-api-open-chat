package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	socketio "github.com/googollee/go-socket.io"
)

type ChatUser struct {
	Username string `json:"userName"`
	SocketID string `json:"socketId"`
}

type Message struct {
	Text     string `json:"text"`
	Name     string `json:"name"`
	ID       string `json:"id"`
	SocketID string `json:"socketID"`
}

func GinMiddleware(allowOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, Content-Length, X-CSRF-Token, Token, session, Origin, Host, Connection, Accept-Encoding, Accept-Language, X-Requested-With")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Request.Header.Del("Origin")

		c.Next()
	}
}

func main() {
	router := gin.New()

	server := socketio.NewServer(nil)

	users := []ChatUser{}

	server.OnConnect("/", func(s socketio.Conn) error {
		log.Printf("⚡: %s user just connected", s.ID())
		server.JoinRoom("/", "chat", s)
		return nil
	})

	server.OnEvent("/", "message", func(s socketio.Conn, msg Message) {
		log.Println("🔥: New message", msg)
		server.BroadcastToRoom("/", "chat", "messageResponse", msg)

	})

	server.OnEvent("/", "typing", func(s socketio.Conn, msg string) {
		server.BroadcastToRoom("/", "chat", "typingResponse", msg)
	})

	server.OnEvent("/", "newUser", func(s socketio.Conn, newUser ChatUser) {
		log.Println("🔥: A new user connected other event", newUser.Username, newUser.SocketID)

		users = append(users, newUser)
		server.BroadcastToRoom("/", "chat", "newUserResponse", users)
	})

	server.OnError("/", func(s socketio.Conn, e error) {
		log.Println("meet error:", e)
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		log.Println("🔥: A user disconnected", reason)
		for i, user := range users {
			if user.SocketID == s.ID() {
				users = append(users[:i], users[i+1:]...)
				break
			}
		}

		server.BroadcastToRoom("/", "chat", "newUserResponse", users)
	})

	go func() {
		if err := server.Serve(); err != nil {
			log.Fatalf("socketio listen error: %s\n", err)
		}
	}()

	defer server.Close()

	router.Use(GinMiddleware("http://localhost:3000"))
	router.GET("/socket.io/*any", gin.WrapH(server))
	router.POST("/socket.io/*any", gin.WrapH(server))

	if err := router.Run(":8000"); err != nil {
		log.Fatal("failed run app: ", err)
	}
}
