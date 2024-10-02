package prism

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeResponse(t *testing.T) {
	assert := assert.New(t)

	call_id := CallId(65532)

	data := "some data"
	got, err := MakeResponse(call_id, TEXT, &data)
	assert.NoError(err)
	assert.Equal("RES\n65532\nTEXT\nsome data", got)

	data = `{ "key": "value" }`
	got, err = MakeResponse(call_id, JSON, &data)
	assert.NoError(err)
	assert.Equal("RES\n65532\nJSON\n{ \"key\": \"value\" }", got)

	expected := "RES\n65532"
	got, err = MakeResponse(call_id, VOID, &data)
	assert.NoError(err)
	assert.Equal(expected, got)

	// make a response with no data
	data = ""
	got, err = MakeResponse(call_id, TEXT, &data)
	assert.NoError(err)
	assert.Equal("RES\n65532\nTEXT\n", got)

	// make response with nil data
	got, err = MakeResponse(call_id, TEXT, nil)
	assert.Error(err)
	assert.Empty(got)
	got, err = MakeResponse(call_id, JSON, nil)
	assert.Error(err)
	assert.Empty(got)

	// make response with data that is too large
	data = strings.Repeat("a", MAX_MESSAGE_SIZE-14)
	got, err = MakeResponse(call_id, TEXT, &data)
	assert.Error(err)
	assert.Empty(got)

	// make response with data that is just small enough
	data = strings.Repeat("a", MAX_MESSAGE_SIZE-15)
	got, err = MakeResponse(call_id, TEXT, &data)
	assert.NoError(err)
	assert.Equal("RES\n65532\nTEXT\n"+data, got)
}

func TestMakeErrorResponse(t *testing.T) {
	assert := assert.New(t)

	call_id := CallId(456)
	got, err := MakeErrorResponse(call_id, "Internal Server Error")
	assert.NoError(err)
	assert.Equal(`ERR
456
Internal Server Error`, got)

	// make an error response with no error message
	got, err = MakeErrorResponse(call_id, "")
	assert.Error(err)
	assert.Empty(got)

	// make an error response with an error message that is too large
	got, err = MakeErrorResponse(call_id, strings.Repeat("a", MAX_MESSAGE_SIZE-7))
	assert.Error(err)
	assert.Empty(got)

	// make an error response with an error message that is just small enough
	s := strings.Repeat("a", MAX_MESSAGE_SIZE-8)
	got, err = MakeErrorResponse(call_id, s)
	assert.Equal("ERR\n456\n"+s, got)
	assert.NoError(err)
}

func TestMakeCastMessage(t *testing.T) {
	assert := assert.New(t)

	topic := "room/123123"

	data := "some data"
	got, err := MakeCastMessage(topic, TEXT, &data)
	assert.Equal("CAST\nroom/123123\nTEXT\nsome data", got)
	assert.NoError(err)

	data = `{ "key": "value" }`
	got, err = MakeCastMessage(topic, JSON, &data)
	assert.Equal("CAST\nroom/123123\nJSON\n{ \"key\": \"value\" }", got)
	assert.NoError(err)

	// message with no data
	got, err = MakeCastMessage(topic, VOID, nil)
	assert.Equal("CAST\nroom/123123", got)
	assert.NoError(err)

	// message with void as type but has data
	data = `some data`
	got, err = MakeCastMessage(topic, VOID, &data)
	assert.Error(err)
	assert.Empty(got)

	// cast message with empty string data
	data = ""
	got, err = MakeCastMessage(topic, TEXT, &data)
	assert.Equal("CAST\nroom/123123\nTEXT\n", got)
	assert.NoError(err)

	// message with nil data
	got, err = MakeCastMessage(topic, TEXT, nil)
	assert.Error(err)
	assert.Empty(got)
	got, err = MakeCastMessage(topic, JSON, nil)
	assert.Error(err)
	assert.Empty(got)

	// make cast message with data that is too large
	data = strings.Repeat("a", MAX_MESSAGE_SIZE-21)
	got, err = MakeCastMessage(topic, TEXT, &data)
	assert.Error(err)
	assert.Empty(got)

	// make cast message with data that is just small enough
	data = strings.Repeat("a", MAX_MESSAGE_SIZE-22)
	got, err = MakeCastMessage(topic, TEXT, &data)
	assert.NoError(err)
	assert.Equal("CAST\nroom/123123\nTEXT\n"+data, got)
}

func TestUnmarshalRequest(t *testing.T) {
	assert := assert.New(t)

	// call
	got, err := UnmarshalRequest([]byte("CALL\n12\nfunc\nTEXT\nhello"))
	assert.NoError(err)
	assert.Equal(CallRequest{
		Request: Request{
			verb:   CALL,
			format: TEXT,
			data:   "hello",
		},
		call_id:  12,
		function: "func",
	}, got)

	// emit
	got, err = UnmarshalRequest([]byte("EMIT\nroom/123123\nTEXT\nhello"))
	assert.NoError(err)
	assert.Equal(EmitRequest{
		Request: Request{verb: EMIT, format: TEXT, data: "hello"},
		event:   "room/123123",
	}, got)

	// empty text data
	got, err = UnmarshalRequest([]byte("CALL\n2\nfunction\nTEXT\n"))
	assert.NoError(err)
	assert.Equal(CallRequest{
		Request:  Request{verb: CALL, format: TEXT, data: ""},
		function: "function",
		call_id:  2,
	}, got)

	// empty text data and no newline
	got, err = UnmarshalRequest([]byte("CALL\n2\nfunction\nTEXT"))
	assert.NoError(err)
	assert.Equal(CallRequest{
		Request:  Request{verb: CALL, format: TEXT, data: ""},
		function: "function",
		call_id:  2,
	}, got)

	// message is too large
	got, err = UnmarshalRequest([]byte("CALL\n" + strings.Repeat("a", MAX_MESSAGE_SIZE)))
	assert.Error(err)
	assert.Nil(got)

	// try some invalid messages
	invalidMessages := []string{
		"hello world",
		"CALL\n0",
		"",
		"CALL\na\nfunc_name",          // invalid call id
		"CALL\n4294967296\nfunc_name", // invalid call id
	}
	for _, message := range invalidMessages {
		got, err := UnmarshalRequest([]byte(message))
		assert.Error(err)
		assert.Empty(got)
	}

	// invalid verb
	got, err = UnmarshalRequest([]byte("INVALID\n0\nfunc\nTEXT\nhello"))
	assert.Error(err)
	assert.Nil(got)

	// invalid format
	got, err = UnmarshalRequest([]byte("CALL\n0\nfunc\nINVALID\nhello"))
	assert.Error(err)
	assert.Nil(got)

	// empty function name
	got, err = UnmarshalRequest([]byte("CALL\n0\n\nTEXT\nhello"))
	assert.Error(err)
	assert.Nil(got)

	// empty event name
	got, err = UnmarshalRequest([]byte("EMIT\n\nTEXT\nhello"))
	assert.Error(err)
	assert.Nil(got)

}
