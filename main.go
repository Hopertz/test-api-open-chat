package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	socketio "github.com/googollee/go-socket.io"
)

func init() {

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	slog.SetDefault(logger)

}

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
		server.JoinRoom("/", "chat", s)
		return nil
	})

	server.OnEvent("/", "message", func(s socketio.Conn, msg Message) {
		server.BroadcastToRoom("/", "chat", "messageResponse", msg)

	})

	server.OnEvent("/", "typing", func(s socketio.Conn, msg string) {
		server.BroadcastToRoom("/", "chat", "typingResponse", msg)
	})

	server.OnEvent("/", "newUser", func(s socketio.Conn, newUser ChatUser) {
		//Check if user already exists and delete old user
		for i, user := range users {
			if user.Username == newUser.Username {
				users = append(users[:i], users[i+1:]...)
				break
			}
		}

		users = append(users, newUser)
		server.BroadcastToRoom("/", "chat", "newUserResponse", users)
	})

	server.OnError("/", func(s socketio.Conn, e error) {
		slog.Error("meet error:", e)
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		slog.Info("closed", reason)
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

	router.Use(GinMiddleware("https://chat.hopertz.me"))
	router.GET("/socket.io/*any", gin.WrapH(server))
	router.POST("/socket.io/*any", gin.WrapH(server))

	if err := router.Run(":8187"); err != nil {
		log.Fatal("failed run app: ", err)
	}
}
