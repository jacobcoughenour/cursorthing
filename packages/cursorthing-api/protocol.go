package main

import (
	"fmt"
	"strconv"
	"strings"
)

const MAX_MESSAGE_SIZE = 4096

type DataFormat int
type CallId uint16

const (
	VOID DataFormat = iota
	JSON
	TEXT
)

type RequestVerb int

const (
	CALL RequestVerb = iota
	EMIT
)

func MakeResponse(call_id CallId, format DataFormat, data string) string {
	switch format {
	case JSON:
		return fmt.Sprintf("RES\n%d\nJSON\n%s", call_id, data)
	case TEXT:
		return fmt.Sprintf("RES\n%d\nTEXT\n%s", call_id, data)
	default:
		return fmt.Sprintf("RES\n%d", call_id)
	}
}

func MakeErrorResponse(call_id CallId, err string) string {
	return fmt.Sprintf("ERR\n%d\n%s", call_id, err)
}

type Request struct {
	verb   RequestVerb
	format DataFormat
	data   string
}

func (r Request) String() string {
	return fmt.Sprintf("Request{verb: %d, format: %d, data: %s}", r.verb, r.format, r.data)
}

type CallRequest struct {
	Request
	call_id CallId
}

type EmitRequest struct {
	Request
	event string
}

// note: this doesn't unmarshal json data
func UnmarshalRequest(message string) (interface{}, error) {

	req := Request{}

	// check if the data is too large
	if len(message) > MAX_MESSAGE_SIZE {
		return req, fmt.Errorf("message too large")
	}

	// split the data into lines
	lines := strings.Split(message, "\n")
	if len(lines) < 2 {
		return req, fmt.Errorf("invalid message")
	}

	// get the data
	if len(lines) > 2 {
		// get format
		formatStr := lines[2]
		if formatStr == "JSON" {
			req.format = JSON
		} else if formatStr == "TEXT" {
			req.format = TEXT
		} else {
			return req, fmt.Errorf("invalid format")
		}

		if len(lines) > 3 {
			// collect the rest of the message as the data
			req.data = strings.Join(lines[3:], "\n")
		} else {
			// format was specified without a trailing new line for data.
			// if you want to send an empty string you need to specify the
			// format as TEXT then have a new line and leave it blank.
			return req, fmt.Errorf("missing data")
		}
	}

	// get the verb
	verbStr := lines[0]
	if verbStr == "CALL" {
		req.verb = CALL
		// get call id
		callIdStr := lines[1]
		// try parsing as a uint16
		callId, err := strconv.ParseUint(callIdStr, 10, 16)
		if err != nil || callId > 65535 {
			return req, fmt.Errorf("invalid call id")
		}
		return interface{}(CallRequest{req, CallId(callId)}), nil

	} else if verbStr == "EMIT" {
		req.verb = EMIT
		// get event
		event := lines[1]
		return interface{}(EmitRequest{req, event}), nil
	}

	return req, fmt.Errorf("invalid verb")
}
