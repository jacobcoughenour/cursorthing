package prism

import (
	"bufio"
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
	call_id  CallId
	function string
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
	if len(message) == 0 {
		return req, fmt.Errorf("empty message")
	}

	scanner := bufio.NewScanner(strings.NewReader(message))
	scanner.Split(bufio.ScanLines)

	// first line
	if !scanner.Scan() {
		return req, fmt.Errorf("missing verb")
	}

	// get the verb
	verbStr := scanner.Text()
	if verbStr == "CALL" {
		req.verb = CALL
	} else if verbStr == "EMIT" {
		req.verb = EMIT
	} else {
		return req, fmt.Errorf("invalid verb")
	}

	// second line
	if !scanner.Scan() {
		if req.verb == CALL {
			return req, fmt.Errorf("missing call id")
		} else if req.verb == EMIT {
			return req, fmt.Errorf("missing event name")
		}
	}

	callId := CallId(0)
	function := ""
	event := ""
	if req.verb == CALL {
		// get call id
		callIdStr := scanner.Text()
		// try parsing as a uint16
		callId, err := strconv.ParseUint(callIdStr, 10, 16)
		if err != nil || callId > 65535 {
			return req, fmt.Errorf("invalid call id")
		}

		// get the function name
		if !scanner.Scan() {
			return req, fmt.Errorf("missing function name")
		}

		function = scanner.Text()
		if function == "" {
			return req, fmt.Errorf("empty function name")
		}

	} else if req.verb == EMIT {
		// get event
		event = scanner.Text()
		if event == "" {
			return req, fmt.Errorf("empty event name")
		}
	}

	// get optional data
	if scanner.Scan() {

		// get format
		formatStr := scanner.Text()
		if formatStr == "JSON" {
			req.format = JSON
		} else if formatStr == "TEXT" {
			req.format = TEXT
		} else {
			return req, fmt.Errorf("invalid format")
		}

		// if there are more lines, then the data is present
		if scanner.Scan() {
			// collect the rest of the message as the data
			req.data = scanner.Text()
			for scanner.Scan() {
				req.data += "\n" + scanner.Text()
			}
		} else {
			// we assume the data is an empty string
			req.data = ""
		}

	}

	if req.verb == CALL {
		return interface{}(CallRequest{req, callId, function}), nil
	}
	// else if req.verb == EMIT {
	return interface{}(EmitRequest{req, event}), nil
	// }
}
