package prism

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

// a wrapper around the gorilla/mux router to implement the prism protocol
type PrismRouter struct {
	*mux.Router
	wg           *sync.WaitGroup
	server       *http.Server
	groups       map[string]*PrismGroup
	funcHandlers map[string]PrismHandlerFunc
}

func NewRouter() *PrismRouter {
	return &PrismRouter{
		mux.NewRouter(),
		nil,
		nil,
		make(map[string]*PrismGroup),
		make(map[string]PrismHandlerFunc),
	}
}

// starts the prism router on the given port.
// this function is not blocking. use Close() to stop the server.
func (s *PrismRouter) ListenAndServe(port int) error {
	// check if already running
	if s.server != nil {
		return fmt.Errorf("server is already running")
	}

	if port == 0 {
		return fmt.Errorf("API port is not set")
	}
	log.Println("Starting API on port", port)

	s.HandleFunc("/ws", s.socketHandler)

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
	})

	s.wg = &sync.WaitGroup{}
	s.wg.Add(1)

	s.server = &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: c.Handler(s)}

	go func() {
		defer s.wg.Done()

		if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
			log.Println("Failed to start server:", err)
		}

		s.server = nil
	}()

	return nil
}

// stops the prism router
func (s *PrismRouter) Close(ctx context.Context) error {
	if s.wg == nil || s.server == nil {
		return fmt.Errorf("server is not running")
	}

	if err := s.server.Shutdown(ctx); err != nil {
		panic(err)
	}
	s.wg.Wait()
	s.wg = nil

	log.Println("Server stopped")
	return nil
}

type PrismClient struct {
	socket *websocket.Conn
	send   chan []byte
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// this is the main handler for the prism protocol
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

			log.Println("Received message:", strings.ReplaceAll(string(b), "\n", " "))

			req, err := UnmarshalRequest(b)
			if err != nil {
				log.Println("Failed to unmarshal message:", err)
				client.send <- []byte("ERR\n\n" + err.Error())
				continue
			}

			// handle request
			switch typedRequest := req.(type) {
			case CallRequest:
				if handler, ok := s.funcHandlers[typedRequest.function]; ok {
					c := newHandlerContext(s, client, typedRequest, r)
					handler(c)
					c.send()
				} else {
					r, err := MakeErrorResponse(typedRequest.call_id, ("function not found: " + typedRequest.function)[0:MAX_MESSAGE_SIZE])
					if err != nil {
						// this should really not happen
						panic(err)
					}
					client.send <- []byte(r)
				}
			case EmitRequest:
				// todo
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

type Context struct {
	router       *PrismRouter
	httpRequest  *http.Request
	param_format DataFormat
	param_data   *string
	client       *PrismClient
	call_id      CallId
	resType      DataFormat
	resData      *string
	resError     error
}

type PrismHandlerFunc func(*Context)

func (s *PrismRouter) HandlePrismFunc(path string, handler PrismHandlerFunc) error {
	if path == "" {
		return fmt.Errorf("path is empty")
	}
	if strings.Contains(path, "\n") {
		return fmt.Errorf("path contains newline")
	}
	if _, ok := s.funcHandlers[path]; ok {
		return fmt.Errorf("handler already exists")
	}

	s.funcHandlers[path] = handler

	return nil
}

func newHandlerContext(router *PrismRouter, client *PrismClient, cr CallRequest, r *http.Request) *Context {
	return &Context{
		router:       router,
		httpRequest:  r,
		param_format: cr.format,
		param_data:   &cr.data,
		client:       client,
		call_id:      cr.call_id,
		resType:      VOID,
		resData:      nil,
		resError:     nil,
	}
}

// get the raw parameter data
func (c *Context) RawParam() (DataFormat, string) {
	return c.param_format, *c.param_data
}

// get the parameter as a string
func (c *Context) TextParam() (string, error) {
	if c.param_format != TEXT {
		return "", fmt.Errorf("parameter is not text")
	}
	return *c.param_data, nil
}

// get the parameter as a string, or nil if it is not present
func (c *Context) OptionalTextParam() (*string, error) {
	if c.param_format == VOID {
		return nil, nil
	} else if c.param_format != TEXT {
		return nil, fmt.Errorf("parameter is not text")
	}
	return c.param_data, nil
}

// get the parameter as json
func (c *Context) JSONParam() (interface{}, error) {
	if c.param_format != JSON {
		return nil, fmt.Errorf("parameter is not json")
	}
	var data interface{}
	err := json.Unmarshal([]byte(*c.param_data), &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// get the parameter as json, or nil if it is not present
func (c *Context) OptionalJSONParam() (*interface{}, error) {
	if c.param_format == VOID {
		return nil, nil
	}
	if c.param_format != JSON {
		return nil, fmt.Errorf("parameter is not json")
	}
	var data interface{}
	err := json.Unmarshal([]byte(*c.param_data), &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// send a response to the client as text
func (c *Context) ResponseText(text string) {
	c.resError = nil
	c.resType = TEXT
	c.resData = &text
}

// send a response to the client as json
func (c *Context) ResponseJSON(data interface{}) {
	b, err := json.Marshal(data)
	if err != nil {
		c.Error(err)
		return
	}
	c.resError = nil
	c.resType = JSON
	resData := string(b)
	c.resData = &resData
}

// send an error to the client
func (c *Context) Error(err error) {
	c.resError = err
	c.resType = VOID
	c.resData = nil
}

// send an error to the client with a formatted message
func (c *Context) Errorf(format string, a ...interface{}) {
	c.Error(fmt.Errorf(format, a...))
}

// sends the response to the client socket
func (c *Context) send() error {
	if c.resError != nil {
		r, err := MakeErrorResponse(c.call_id, c.resError.Error())
		if err != nil {
			return err
		}
		c.client.send <- []byte(r)
		return nil
	}
	switch c.resType {
	case JSON:
		r, err := MakeResponse(c.call_id, JSON, c.resData)
		if err != nil {
			return err
		}
		c.client.send <- []byte(r)
	case TEXT:
		r, err := MakeResponse(c.call_id, TEXT, c.resData)
		if err != nil {
			return err
		}
		c.client.send <- []byte(r)
	default:
		r, err := MakeResponse(c.call_id, VOID, nil)
		if err != nil {
			return err
		}
		c.client.send <- []byte(r)
	}
	return nil
}

// grouping stuff

func (c *Context) getOrCreateGroup(groupName string) *PrismGroup {
	group, ok := c.router.groups[groupName]
	if !ok {
		group = &PrismGroup{
			name:       groupName,
			router:     c.router,
			clients:    make(map[*PrismClient]bool),
			broadcast:  make(chan []byte),
			register:   make(chan *PrismClient),
			unregister: make(chan *PrismClient),
		}
		c.router.groups[groupName] = group
		go group.run()
	}
	return group
}

func (c *Context) AddToGroup(groupName string) error {
	group := c.getOrCreateGroup(groupName)
	// is the client already in the group?
	if _, ok := group.clients[c.client]; ok {
		// do nothing
		return fmt.Errorf("client is already in group")
	}
	group.register <- c.client
	return nil
}

func (c *Context) IsInGroup(groupName string) bool {
	group, ok := c.router.groups[groupName]
	if !ok {
		return false
	}
	_, ok = group.clients[c.client]
	return ok
}

func (c *Context) RemoveFromGroup(groupName string) error {
	group, err := c.router.groups[groupName]
	if err {
		return fmt.Errorf("group not found")
	}
	// is the client in the group?
	if _, ok := group.clients[c.client]; !ok {
		// do nothing
		return fmt.Errorf("client is not in group")
	}
	group.unregister <- c.client
	return nil
}

func (c *Context) GroupHasUsers(groupName string) bool {
	_, ok := c.router.groups[groupName]
	return ok
}

func (c *Context) BroadcastTextToGroup(groupName string, text string) error {
	group, ok := c.router.groups[groupName]
	if !ok {
		return nil
	}
	r, err := MakeCastMessage(groupName, TEXT, &text)
	if err != nil {
		return err
	}
	group.broadcast <- []byte(r)
	return nil
}

func (c *Context) BroadcastJSONToGroup(groupName string, data interface{}) error {
	group, ok := c.router.groups[groupName]
	if !ok {
		return nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	s := string(b)
	r, err := MakeCastMessage(groupName, JSON, &s)
	if err != nil {
		return err
	}
	group.broadcast <- []byte(r)
	return nil
}

func (c *Context) BroadcastVoidToGroup(groupName string, text string) error {
	group := c.getOrCreateGroup(groupName)
	r, err := MakeCastMessage(groupName, TEXT, nil)
	if err != nil {
		return err
	}
	group.broadcast <- []byte(r)
	return nil
}

// a group of clients that can be broadcasted to
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

// broadcast group channel event loop
func (g *PrismGroup) run() {
	for {
		select {
		// add client to group
		case client := <-g.register:
			g.clients[client] = true
			fmt.Println("client joined group:", g.name)
		// remove client from group
		case client := <-g.unregister:
			if _, ok := g.clients[client]; ok {
				fmt.Println("client unsubscribed from group:", g.name)
				g.router = nil
				delete(g.clients, client)
			}
		// broadcast message to all clients
		case message := <-g.broadcast:
			for client := range g.clients {
				client.send <- message
			}
		}
	}
}
