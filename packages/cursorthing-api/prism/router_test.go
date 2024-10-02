package prism

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestPrismRouter(t *testing.T) {
	assert := assert.New(t)

	router := NewRouter()
	assert.NotNil(router, "Expected a new router, got nil")

	router.ListenAndServe(8080)
	if router != nil && router.wg == nil {
		t.Error("Expected a non-nil waitgroup, got nil")
	}

	// try to start the server twice
	err := router.ListenAndServe(8080)
	assert.Error(err)

	err = router.Close(context.Background())
	assert.NoError(err)
	if router != nil && router.wg != nil {
		t.Error("Expected a nil waitgroup, got", router.wg)
	}
	assert.Nil(router.server, "Expected a nil server, got", router.server)

	// try to close the server twice
	err = router.Close(context.Background())
	assert.Error(err)
}

func TestRestEndpoint(t *testing.T) {
	assert := assert.New(t)

	router := NewRouter()
	router.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	router.ListenAndServe(8080)

	res, err := http.Get("http://localhost:8080/status")
	assert.NoError(err)
	assert.NotNil(res)
	assert.Equal(200, res.StatusCode)

	buf := make([]byte, 2)
	n, err := res.Body.Read(buf)
	assert.Equal(io.EOF, err)
	assert.Equal(2, n)
	assert.Equal("OK", string(buf))

	err = router.Close(context.Background())
	assert.NoError(err)
}

func TestWebsocketConnection(t *testing.T) {

	router := NewRouter()
	router.ListenAndServe(8080)

	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		t.Error(err)
	}

	conn.Close()

	err = router.Close(context.Background())
	assert.NoError(t, err)
}

func TestPrismFuncCall(t *testing.T) {
	assert := assert.New(t)

	router := NewRouter()

	voidHandler := func(c *Context) {}

	expected := 0

	assert.NoError(router.HandlePrismFunc("my_func", voidHandler))
	expected++

	// try adding the same handler twice
	assert.Error(router.HandlePrismFunc("my_func", voidHandler))
	// try adding a handler with an invalid name
	assert.Error(router.HandlePrismFunc("", voidHandler))
	// try adding a handler with an invalid name
	assert.Error(router.HandlePrismFunc("my_func\n", voidHandler))

	assert.NoError(router.HandlePrismFunc("my_func_ok", func(c *Context) {
		c.ResponseText("OK")
	}))
	expected++

	assert.NoError(router.HandlePrismFunc("my_func_err", func(c *Context) {
		c.Errorf("error")
	}))
	expected++

	assert.NoError(router.HandlePrismFunc("my_func_json", func(c *Context) {
		c.ResponseJSON(map[string]string{"key": "value"})
	}))
	expected++

	assert.NoError(router.HandlePrismFunc(("echo_required_text_param"), func(c *Context) {
		param, err := c.TextParam()
		if err != nil {
			c.Error(err)
			return
		}
		c.ResponseText(param)
	}))
	expected++

	assert.NoError(router.HandlePrismFunc(("echo_optional_text_param"), func(c *Context) {
		param, err := c.OptionalTextParam()
		if err != nil {
			c.Error(err)
			return
		}
		if param == nil {
			c.ResponseText("nil")
			return
		}
		c.ResponseText(*param)
	}))
	expected++

	assert.NoError(router.HandlePrismFunc(("echo_required_json_param"), func(c *Context) {
		param, err := c.JSONParam()
		if err != nil {
			c.Error(err)
			return
		}
		c.ResponseJSON(param)
	}))
	expected++

	assert.NoError(router.HandlePrismFunc(("echo_optional_json_param"), func(c *Context) {
		param, err := c.OptionalJSONParam()
		if err != nil {
			c.Error(err)
			return
		}
		if param == nil {
			c.ResponseText("nil")
			return
		}
		c.ResponseJSON(*param)
	}))
	expected++

	assert.Equal(expected, len(router.funcHandlers))

	router.ListenAndServe(8080)

	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	assert.NoError(err)

	// test void return
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n0\nmy_func"))
	_, message, err := conn.ReadMessage()
	assert.NoError(err)
	assert.Equal("RES\n0", string(message))

	// test text return
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n1\nmy_func_ok"))
	_, message, err = conn.ReadMessage()
	assert.NoError(err)
	assert.Equal("RES\n1\nTEXT\nOK", string(message))

	// test error return
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n2\nmy_func_err"))
	_, message, err = conn.ReadMessage()
	assert.NoError(err)
	assert.Equal("ERR\n2\nerror", string(message))

	// test text param and text return
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n3\necho_required_text_param\nTEXT\nhello\nworld"))
	_, message, err = conn.ReadMessage()
	assert.NoError(err)
	assert.Equal("RES\n3\nTEXT\nhello\nworld", string(message))

	// test sending void param to required text param handler
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n4\necho_required_text_param"))
	_, message, err = conn.ReadMessage()
	assert.NoError(err)
	assert.Equal("ERR\n4\nparameter is not text", string(message))

	// test sending json param to required text param handler
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n5\necho_required_text_param\nJSON\n{\"key\":\"value\"}"))
	_, message, err = conn.ReadMessage()
	assert.NoError(err)
	assert.Equal("ERR\n5\nparameter is not text", string(message))

	// test sending void param to optional text param handler
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n6\necho_optional_text_param"))
	_, message, err = conn.ReadMessage()
	assert.NoError(err)
	assert.Equal("RES\n6\nTEXT\nnil", string(message))

	// test sending text param to optional text param handler
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n7\necho_optional_text_param\nTEXT\nhello\nworld\n"))
	_, message, err = conn.ReadMessage()
	assert.NoError(err)
	assert.Equal("RES\n7\nTEXT\nhello\nworld", string(message))

	// test sending json param to optional text param handler
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n8\necho_optional_text_param\nJSON\n{\"key\":\"value\"}"))
	_, message, err = conn.ReadMessage()
	assert.NoError(err)
	assert.Equal("ERR\n8\nparameter is not text", string(message))

	// test sending void param to required json param handler
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n9\necho_required_json_param"))
	_, message, err = conn.ReadMessage()
	assert.NoError(err)
	assert.Equal("ERR\n9\nparameter is not json", string(message))

	// test sending text param to required json param handler
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n10\necho_required_json_param\nTEXT\nhello\nworld\n"))
	_, message, err = conn.ReadMessage()
	assert.NoError(err)
	assert.Equal("ERR\n10\nparameter is not json", string(message))

	// test sending json param to required json param handler
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n11\necho_required_json_param\nJSON\n{\"key\":\"value\"}"))
	_, message, err = conn.ReadMessage()
	assert.NoError(err)
	assert.Equal("RES\n11\nJSON\n{\"key\":\"value\"}", string(message))

	// test sending void param to optional json param handler
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n12\necho_optional_json_param"))
	_, message, err = conn.ReadMessage()
	assert.NoError(err)
	assert.Equal("RES\n12\nTEXT\nnil", string(message))

	// test sending text param to optional json param handler
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n13\necho_optional_json_param\nTEXT\nhello\nworld\n"))
	_, message, err = conn.ReadMessage()
	assert.NoError(err)
	assert.Equal("ERR\n13\nparameter is not json", string(message))

	// test sending json param to optional json param handler
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n14\necho_optional_json_param\nJSON\n{\"key\":\"value\"}"))
	_, message, err = conn.ReadMessage()
	assert.NoError(err)
	assert.Equal("RES\n14\nJSON\n{\"key\":\"value\"}", string(message))

	conn.Close()

	err = router.Close(context.Background())
	assert.NoError(err)
}

func TestPrismGroups(t *testing.T) {
	assert := assert.New(t)

	router := NewRouter()

	assert.NoError(router.HandlePrismFunc("join", func(c *Context) {
		groupName, err := c.TextParam()
		if err != nil {
			c.Error(err)
			return
		}
		err = c.AddToGroup(groupName)
		if err != nil {
			c.Error(err)
			return
		}
	}))

	assert.NoError(router.HandlePrismFunc("leave", func(c *Context) {
		groupName, err := c.TextParam()
		if err != nil {
			c.Error(err)
			return
		}
		err = c.RemoveFromGroup(groupName)
		if err != nil {
			c.Error(err)
			return
		}
	}))

	router.ListenAndServe(8080)

	// open first connection
	conn1, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	assert.NoError(err)

	// test joining a group
	conn1.WriteMessage(websocket.TextMessage, []byte("CALL\n0\njoin\nTEXT\ngroup1"))
	_, message, err := conn1.ReadMessage()
	assert.NoError(err)
	assert.Equal("RES\n0", string(message))

	// open second connection
	conn2, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	assert.NoError(err)

}
