package main

import (
	"testing"
)

func TestMakeResponse(t *testing.T) {
	call_id := CallId(65532)

	format := TEXT
	data := "some data"
	expected := "RES\n65532\nTEXT\nsome data"
	if got := MakeResponse(call_id, format, data); got != expected {
		t.Errorf("MakeResponse(%v, %v, %v) = %v; want %v", call_id, format, data, got, expected)
	}

	format = JSON
	data = `{ "key": "value" }`
	expected = "RES\n65532\nJSON\n{ \"key\": \"value\" }"
	if got := MakeResponse(call_id, format, data); got != expected {
		t.Errorf("MakeResponse(%v, %v, %v) = %v; want %v", call_id, format, data, got, expected)
	}

	format = VOID
	expected = "RES\n65532"
	if got := MakeResponse(call_id, format, data); got != expected {
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
	message := "CALL\n0\nTEXT\nhello"
	expected := CallRequest{
		Request: Request{
			verb:   CALL,
			format: TEXT,
			data:   "hello",
		},
		call_id: 0,
	}
	if got, err := UnmarshalRequest(message); err != nil || got != expected {
		t.Errorf("UnmarshalRequest(%v) = %v, %v; want %v, nil", message, got, err, expected)
	}

	message = "EMIT\nroom/123123\nTEXT\nhello"
	expected2 := EmitRequest{
		Request: Request{verb: EMIT, format: TEXT, data: "hello"},
		event:   "room/123123",
	}
	if got, err := UnmarshalRequest(message); err != nil || got != expected2 {
		t.Errorf("UnmarshalRequest(%v) = %v, %v; want %v, nil", message, got, err, expected2)
	}
}
