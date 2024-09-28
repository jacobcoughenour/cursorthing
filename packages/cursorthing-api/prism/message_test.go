package prism

import (
	"strings"
	"testing"
)

func TestMakeResponse(t *testing.T) {
	call_id := CallId(65532)

	format := TEXT
	data := "some data"
	expected := "RES\n65532\nTEXT\nsome data"
	if got := MakeResponse(call_id, format, &data); got != expected {
		t.Errorf("MakeResponse(%v, %v, %v) = %v; want %v", call_id, format, data, got, expected)
	}

	format = JSON
	data = `{ "key": "value" }`
	expected = "RES\n65532\nJSON\n{ \"key\": \"value\" }"
	if got := MakeResponse(call_id, format, &data); got != expected {
		t.Errorf("MakeResponse(%v, %v, %v) = %v; want %v", call_id, format, data, got, expected)
	}

	format = VOID
	expected = "RES\n65532"
	if got := MakeResponse(call_id, format, &data); got != expected {
		t.Errorf("MakeResponse(%v, %v, %v) = %v; want %v", call_id, format, data, got, expected)
	}
}

func TestMakeErrorResponse(t *testing.T) {
	call_id := CallId(456)
	err := "Internal Server Error"
	expected := `ERR
456
Internal Server Error`
	if got := MakeErrorResponse(call_id, err); got != expected {
		t.Errorf("MakeErrorResponse(%v, %v) = %v; want %v", call_id, err, got, expected)
	}
}

func TestUnmarshalRequest(t *testing.T) {
	message := []byte("CALL\n12\nfunc\nTEXT\nhello")
	expected := CallRequest{
		Request: Request{
			verb:   CALL,
			format: TEXT,
			data:   "hello",
		},
		call_id:  12,
		function: "func",
	}
	if got, err := UnmarshalRequest(message); err != nil || got != expected {
		t.Errorf("UnmarshalRequest(%v) = %v, %v; want %v, nil", message, got, err, expected)
	}

	message = []byte("EMIT\nroom/123123\nTEXT\nhello")
	expected2 := EmitRequest{
		Request: Request{verb: EMIT, format: TEXT, data: "hello"},
		event:   "room/123123",
	}
	if got, err := UnmarshalRequest(message); err != nil || got != expected2 {
		t.Errorf("UnmarshalRequest(%v) = %v, %v; want %v, nil", message, got, err, expected2)
	}

	// test with empty text data
	message = []byte("CALL\n0\nfunction\nTEXT\n")
	expected3 := CallRequest{
		Request:  Request{verb: CALL, format: TEXT, data: ""},
		function: "function",
		call_id:  0,
	}
	if got, err := UnmarshalRequest(message); err != nil || got != expected3 {
		t.Errorf("UnmarshalRequest(%v) = %v, %v; want %v, nil", message, got, err, expected3)
	}

	// test with empty text data and no newline
	message = []byte("CALL\n0\nfunction\nTEXT")
	if got, err := UnmarshalRequest(message); err != nil || got != expected3 {
		t.Errorf("UnmarshalRequest(%v) = %v, %v; want %v, nil", message, got, err, expected3)
	}

	// make a message that is too large
	message = []byte("CAL\n" + strings.Repeat("a", MAX_MESSAGE_SIZE))
	if _, err := UnmarshalRequest(message); err == nil {
		t.Errorf("UnmarshalRequest(%v) = _, nil; want _, error", message)
	}

	// try some invalid messages
	invalidMessages := []string{
		"hello world",
		"CALL\n0",
		"",
		"CALL\na\nfunc_name",          // invalid call id
		"CALL\n4294967296\nfunc_name", // invalid call id
	}
	for _, message := range invalidMessages {
		_, err := UnmarshalRequest([]byte(message))
		if err == nil {
			t.Errorf("UnmarshalRequest(%v) = _, nil; want _, error", message)
		}
	}

	// invalid verb
	message = []byte("INVALID\n0\nfunc\nTEXT\nhello")
	if _, err := UnmarshalRequest(message); err == nil {
		t.Errorf("UnmarshalRequest(%v) = _, nil; want _, error", message)
	}

	// invalid format
	message = []byte("CALL\n0\nfunc\nINVALID\nhello")
	if _, err := UnmarshalRequest(message); err == nil {
		t.Errorf("UnmarshalRequest(%v) = _, nil; want _, error", message)
	}

	// empty function name
	message = []byte("CALL\n0\n\nTEXT\nhello")
	if _, err := UnmarshalRequest(message); err == nil {
		t.Errorf("UnmarshalRequest(%v) = _, nil; want _, error", message)
	}

	// empty event name
	message = []byte("EMIT\n\nTEXT\nhello")
	if _, err := UnmarshalRequest(message); err == nil {
		t.Errorf("UnmarshalRequest(%v) = _, nil; want _, error", message)
	}

}
