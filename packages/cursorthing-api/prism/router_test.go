package prism

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/gorilla/websocket"
)

func TestPrismRouter(t *testing.T) {

	router := NewRouter()

	if router == nil {
		t.Error("Expected a new router, got nil")
	}

	router.ListenAndServe(8080)
	if router == nil {
		t.Error("Expected a new router, got nil")
	}
	if router != nil && router.wg == nil {
		t.Error("Expected a waitgroup, got nil")
	}

	// try to start the server twice
	err := router.ListenAndServe(8080)
	if err == nil {
		t.Error("Expected an error, got nil")
	}

	err = router.Close(context.Background())
	if err != nil {
		t.Error("Expected no error, got", err)
	}
	if router == nil {
		t.Error("Expected a new router, got nil")
	}
	if router != nil && router.wg != nil {
		t.Error("Expected a nil waitgroup, got", router.wg)
	}
	if router.server != nil {
		t.Error("Expected a nil server, got", router.server)
	}

	// try to close the server twice
	err = router.Close(context.Background())
	if err == nil {
		t.Error("Expected an error, got nil")
	}

}

func TestRestEndpoint(t *testing.T) {

	router := NewRouter()

	router.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	router.ListenAndServe(8080)

	res, err := http.Get("http://localhost:8080/status")
	if err != nil {
		t.Error(err)
	}

	if res.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", res.StatusCode)
	}

	buf := make([]byte, 2)
	n, err := res.Body.Read(buf)
	if err != io.EOF {
		t.Error(err)
	}

	if n != 2 {
		t.Errorf("Expected 2 bytes, got %d", n)
	}

	if string(buf) != "OK" {
		t.Errorf("Expected 'OK', got '%s'", string(buf))
	}

	err = router.Close(context.Background())
	if err != nil {
		t.Error("Expected no error, got", err)
	}
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
	if err != nil {
		t.Error("Expected no error, got", err)
	}
}

func TestPrismFuncCall(t *testing.T) {

	router := NewRouter()

	voidHandler := func(c *Context) {}

	expected := 0

	err := router.HandlePrismFunc("my_func", voidHandler)
	if err != nil {
		t.Error("Expected no error, got", err)
	}
	expected++

	// try adding the same handler twice
	err = router.HandlePrismFunc("my_func", voidHandler)
	if err == nil {
		t.Error("Expected an error, got nil")
	}
	// try adding a handler with an invalid name
	err = router.HandlePrismFunc("", voidHandler)
	if err == nil {
		t.Error("Expected an error, got nil")
	}
	// try adding a handler with an invalid name
	err = router.HandlePrismFunc("my_func\n", voidHandler)
	if err == nil {
		t.Error("Expected an error, got nil")
	}

	err = router.HandlePrismFunc("my_func_ok", func(c *Context) {
		c.ResponseText("OK")
	})
	if err != nil {
		t.Error("Expected no error, got", err)
	}
	expected++

	err = router.HandlePrismFunc("my_func_err", func(c *Context) {
		c.Errorf("error")
	})
	if err != nil {
		t.Error("Expected no error, got", err)
	}
	expected++

	err = router.HandlePrismFunc("my_func_json", func(c *Context) {
		c.ResponseJSON(map[string]string{"key": "value"})
	})
	if err != nil {
		t.Error("Expected no error, got", err)
	}
	expected++

	err = router.HandlePrismFunc(("echo_required_text_param"), func(c *Context) {
		param, err := c.TextParam()
		if err != nil {
			c.Error(err)
			return
		}
		c.ResponseText(param)
	})
	if err != nil {
		t.Error("Expected no error, got", err)
	}
	expected++

	err = router.HandlePrismFunc(("echo_optional_text_param"), func(c *Context) {
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
	})
	if err != nil {
		t.Error("Expected no error, got", err)
	}
	expected++

	err = router.HandlePrismFunc(("echo_required_json_param"), func(c *Context) {
		param, err := c.JSONParam()
		if err != nil {
			c.Error(err)
			return
		}
		c.ResponseJSON(param)
	})
	if err != nil {
		t.Error("Expected no error, got", err)
	}
	expected++

	err = router.HandlePrismFunc(("echo_optional_json_param"), func(c *Context) {
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
	})
	if err != nil {
		t.Error("Expected no error, got", err)
	}
	expected++

	if len(router.funcHandlers) != expected {
		t.Errorf("Expected %d handlers, got %d", expected, len(router.funcHandlers))
	}

	router.ListenAndServe(8080)

	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		t.Error(err)
	}

	// test void return
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n0\nmy_func"))
	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Error(err)
	}
	if string(message) != "RES\n0" {
		t.Errorf("Expected 'RES\n0', got '%s'", string(message))
	}

	// test text return
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n1\nmy_func_ok"))
	_, message, err = conn.ReadMessage()
	if err != nil {
		t.Error(err)
	}
	if string(message) != "RES\n1\nTEXT\nOK" {
		t.Errorf("Expected 'RES\n1\nTEXT\nOK', got '%s'", string(message))
	}

	// test error return
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n2\nmy_func_err"))
	_, message, err = conn.ReadMessage()
	if err != nil {
		t.Error(err)
	}
	if string(message) != "ERR\n2\nerror" {
		t.Errorf("Expected 'ERR\n2\nerror', got '%s'", string(message))
	}

	// test text param and text return
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n3\necho_required_text_param\nTEXT\nhello\nworld"))
	_, message, err = conn.ReadMessage()
	if err != nil {
		t.Error(err)
	}
	if string(message) != "RES\n3\nTEXT\nhello\nworld" {
		t.Errorf("Expected 'RES\n3\nTEXT\nhello\nworld', got '%s'", string(message))
	}

	// test sending void param to required text param handler
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n4\necho_required_text_param"))
	_, message, err = conn.ReadMessage()
	if err != nil {
		t.Error(err)
	}
	if string(message) != "ERR\n4\nparameter is not text" {
		t.Errorf("Expected 'ERR\n4\nparameter is not text', got '%s'", string(message))
	}

	// test sending json param to required text param handler
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n5\necho_required_text_param\nJSON\n{\"key\":\"value\"}"))
	_, message, err = conn.ReadMessage()
	if err != nil {
		t.Error(err)
	}
	if string(message) != "ERR\n5\nparameter is not text" {
		t.Errorf("Expected 'ERR\n5\nparameter is not text', got '%s'", string(message))
	}

	// test sending void param to optional text param handler
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n6\necho_optional_text_param"))
	_, message, err = conn.ReadMessage()
	if err != nil {
		t.Error(err)
	}
	if string(message) != "RES\n6\nTEXT\nnil" {
		t.Errorf("Expected 'RES\n6\nTEXT\nnil', got '%s'", string(message))
	}

	// test sending text param to optional text param handler
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n7\necho_optional_text_param\nTEXT\nhello\nworld\n"))
	_, message, err = conn.ReadMessage()
	if err != nil {
		t.Error(err)
	}
	if string(message) != "RES\n7\nTEXT\nhello\nworld" {
		t.Errorf("Expected 'RES\n7\nTEXT\nhello\nworld', got '%s'", string(message))
	}

	// test sending json param to optional text param handler
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n8\necho_optional_text_param\nJSON\n{\"key\":\"value\"}"))
	_, message, err = conn.ReadMessage()
	if err != nil {
		t.Error(err)
	}
	if string(message) != "ERR\n8\nparameter is not text" {
		t.Errorf("Expected 'ERR\n8\nparameter is not text', got '%s'", string(message))
	}

	// test sending void param to required json param handler
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n9\necho_required_json_param"))
	_, message, err = conn.ReadMessage()
	if err != nil {
		t.Error(err)
	}
	if string(message) != "ERR\n9\nparameter is not json" {
		t.Errorf("Expected 'ERR\n9\nparameter is not json', got '%s'", string(message))
	}

	// test sending text param to required json param handler
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n10\necho_required_json_param\nTEXT\nhello\nworld\n"))
	_, message, err = conn.ReadMessage()
	if err != nil {
		t.Error(err)
	}
	if string(message) != "ERR\n10\nparameter is not json" {
		t.Errorf("Expected 'ERR\n10\nparameter is not json', got '%s'", string(message))
	}

	// test sending json param to required json param handler
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n11\necho_required_json_param\nJSON\n{\"key\":\"value\"}"))
	_, message, err = conn.ReadMessage()
	if err != nil {
		t.Error(err)
	}
	if string(message) != "RES\n11\nJSON\n{\"key\":\"value\"}" {
		t.Errorf("Expected 'RES\n11\nJSON\n{\"key\":\"value\"}', got '%s'", string(message))
	}

	// test sending void param to optional json param handler
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n12\necho_optional_json_param"))
	_, message, err = conn.ReadMessage()
	if err != nil {
		t.Error(err)
	}
	if string(message) != "RES\n12\nTEXT\nnil" {
		t.Errorf("Expected 'RES\n12\nTEXT\nnil', got '%s'", string(message))
	}

	// test sending text param to optional json param handler
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n13\necho_optional_json_param\nTEXT\nhello\nworld\n"))
	_, message, err = conn.ReadMessage()
	if err != nil {
		t.Error(err)
	}
	if string(message) != "ERR\n13\nparameter is not json" {
		t.Errorf("Expected 'ERR\n13\nparameter is not json', got '%s'", string(message))
	}

	// test sending json param to optional json param handler
	conn.WriteMessage(websocket.TextMessage, []byte("CALL\n14\necho_optional_json_param\nJSON\n{\"key\":\"value\"}"))
	_, message, err = conn.ReadMessage()
	if err != nil {
		t.Error(err)
	}
	if string(message) != "RES\n14\nJSON\n{\"key\":\"value\"}" {
		t.Errorf("Expected 'RES\n14\nJSON\n{\"key\":\"value\"}', got '%s'", string(message))
	}

	conn.Close()

	err = router.Close(context.Background())
	if err != nil {
		t.Error("Expected no error, got", err)
	}
}
