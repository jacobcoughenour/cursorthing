package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

type Server struct {
	groups map[string]*Group
}

func NewServer() *Server {
	return &Server{}
}

// wrapper around the websocket connection for tracking our current connections
type Client struct {
	socket *websocket.Conn
	send   chan []byte
}

// a collection of clients that can be broadcast to
type Group struct {
	server *Server
	url    string
	// map of what clients are in this group
	clients map[*Client]bool
	// channel to broadcast messages to all clients
	broadcast chan []byte
	// channels to register and unregister clients
	register   chan *Client
	unregister chan *Client
}

func NewGroup(server *Server, url string) *Group {
	return &Group{
		server:     server,
		url:        url,
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (server *Server) ListenAndServe(port int) error {

	if port == 0 {
		return fmt.Errorf("API port is not set")
	}

	log.Println("Starting API on port", port)

	router := mux.NewRouter()

	router.HandleFunc("/ws", server.socketHandler)
	router.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
	})

	go server.handleEvents()

	http.ListenAndServe(fmt.Sprintf(":%d", port), c.Handler(router))

	return nil
}

func (server *Server) handleEvents() {

}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (s *Server) socketHandler(w http.ResponseWriter, r *http.Request) {
	// upgrade to a websocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade connection:", err)
		return
	}

	// create client
	client := &Client{
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
			switch msg["operation"] {
			case "group/join":
				{
					// groupString := msg["url"].(string)
					// s.getOrCreateGroup(topicString).register <- client
				}
			case "group/leave":
				{
					topicString := msg["url"].(string)
					topic, ok := s.groups[topicString]
					if !ok {
						log.Println("Failed to get topic group:", topicString)
						continue
					}
					topic.unregister <- client
					// s.deleteGroupIfEmpty(topicString)
				}
			}
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

func (server *Server) normalizeUrl(url string) string {
	// todo
	return url
}

func (server *Server) getOrCreateTopic(url string) *Group {
	if _, ok := server.groups[url]; !ok {
		server.groups[url] = NewGroup(server, url)
		fmt.Println("created new topic group:", url)
	}
	return server.groups[url]
}

func (s *Server) deleteGroupIfEmpty(url string) {
	group, ok := s.groups[url]
	if ok && len(group.clients) == 0 {
		delete(s.groups, url)
	}
}
