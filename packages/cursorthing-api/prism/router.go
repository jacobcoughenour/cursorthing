package prism

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

// a wrapper around the gorilla/mux router
// to implement the prism protocol
type PrismRouter struct {
	*mux.Router
	groups map[string]*PrismGroup
}

func NewRouter() *PrismRouter {
	return &PrismRouter{
		mux.NewRouter(),
		make(map[string]*PrismGroup),
	}
}

func (s *PrismRouter) ListenAndServe(port int) error {
	if port == 0 {
		return fmt.Errorf("API port is not set")
	}
	log.Println("Starting API on port", port)

	s.HandleFunc("/ws", s.socketHandler)

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
	})

	http.ListenAndServe(fmt.Sprintf(":%d", port), c.Handler(s))

	return nil
}

type PrismClient struct {
	socket *websocket.Conn
	send   chan []byte
}

type PrismGroup struct {
	// name of the group
	name string
	// reference to the parent router
	router *PrismRouter
	// what clients are in this group
	clients map[*PrismClient]bool
	// channel to broadcast messages to all clients
	broadcast chan []byte
	// channels to register and unregister clients
	register   chan *PrismClient
	unregister chan *PrismClient
}

func addGroup(router *PrismRouter, name string) *PrismGroup {
	return &PrismGroup{
		name:       name,
		router:     router,
		clients:    make(map[*PrismClient]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *PrismClient),
		unregister: make(chan *PrismClient),
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (s *PrismRouter) socketHandler(w http.ResponseWriter, r *http.Request) {
	// upgrade to a websocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade connection:", err)
		return
	}

	// create client
	client := &PrismClient{
		socket: conn,
		send:   make(chan []byte),
	}

	// reader goroutine
	go func() {
		defer func() {
			// unregister client from all groups
			for _, group := range s.groups {
				group.unregister <- client
			}
			conn.Close()
			close(client.send)
		}()

		for {
			_, b, err := conn.ReadMessage()
			if err != nil {
				// unregister client from all groups
				for _, group := range s.groups {
					group.unregister <- client
					// s.deleteGroupIfEmpty(key)
				}
				conn.Close()
				break
			}

			log.Println("Received message:", string(b))

			// json parse message
			var msg map[string]interface{}
			err = json.Unmarshal(b, &msg)
			if err != nil {
				log.Println("Failed to unmarshal message:", err)
				continue
			}

			// handle message
			// todo
		}
	}()

	// writer goroutine
	go func() {
		defer conn.Close()
		for {
			message, ok := <-client.send
			if !ok {
				conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			conn.WriteMessage(websocket.TextMessage, message)
		}
	}()
}
