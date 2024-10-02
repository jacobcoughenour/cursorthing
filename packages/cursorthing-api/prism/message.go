package prism

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

// this is in character length
const MAX_MESSAGE_SIZE = 4096

type DataFormat int
type CallId uint32

const MAX_CALL_ID = 0xFFFFFFFF

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

func MakeResponse(call_id CallId, format DataFormat, data *string) (string, error) {
	if format != VOID && data == nil {
		return "", fmt.Errorf("missing data")
	}
	message := ""
	switch format {
	case JSON:
		message = fmt.Sprintf("RES\n%d\nJSON\n%s", call_id, *data)
	case TEXT:
		message = fmt.Sprintf("RES\n%d\nTEXT\n%s", call_id, *data)
	default:
		message = fmt.Sprintf("RES\n%d", call_id)
	}
	if len(message) > MAX_MESSAGE_SIZE {
		return "", fmt.Errorf("message too large")
	}
	return message, nil
}

func MakeErrorResponse(call_id CallId, err string) (string, error) {
	if len(err) == 0 {
		return "", fmt.Errorf("empty error message")
	}
	message := fmt.Sprintf("ERR\n%d\n%s", call_id, err)
	if len(message) > MAX_MESSAGE_SIZE {
		return "", fmt.Errorf("message too large")
	}
	return message, nil
}

func MakeCastMessage(topic string, format DataFormat, data *string) (string, error) {
	if format != VOID && data == nil {
		return "", fmt.Errorf("missing data")
	} else if format == VOID && data != nil {
		return "", fmt.Errorf("data present but format is void")
	}
	message := ""
	switch format {
	case JSON:
		message = fmt.Sprintf("CAST\n%s\nJSON\n%s", topic, *data)
	case TEXT:
		message = fmt.Sprintf("CAST\n%s\nTEXT\n%s", topic, *data)
	default:
		message = fmt.Sprintf("CAST\n%s", topic)
	}
	if len(message) > MAX_MESSAGE_SIZE {
		return "", fmt.Errorf("message too large")
	}
	return message, nil
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
func UnmarshalRequest(data []byte) (interface{}, error) {
	req := Request{}

	message := string(data)

	// check if the data is too large
	if len(message) > MAX_MESSAGE_SIZE {
		return nil, fmt.Errorf("message too large")
	}
	if len(message) == 0 {
		return nil, fmt.Errorf("empty message")
	}

	scanner := bufio.NewScanner(strings.NewReader(message))
	scanner.Split(bufio.ScanLines)

	// first line
	if !scanner.Scan() {
		return nil, fmt.Errorf("missing verb")
	}

	// get the verb
	verbStr := scanner.Text()
	if verbStr == "CALL" {
		req.verb = CALL
	} else if verbStr == "EMIT" {
		req.verb = EMIT
	} else {
		return nil, fmt.Errorf("invalid verb")
	}

	// second line
	if !scanner.Scan() {
		if req.verb == CALL {
			return nil, fmt.Errorf("missing call id")
		} else if req.verb == EMIT {
			return nil, fmt.Errorf("missing event name")
		}
	}

	callId := CallId(0)
	function := ""
	event := ""
	if req.verb == CALL {
		// get call id
		callIdStr := scanner.Text()
		// try parsing as a uint32
		callIdInt, err := strconv.ParseUint(callIdStr, 10, 32)
		if err != nil || callIdInt > MAX_CALL_ID {
			return nil, fmt.Errorf("invalid call id")
		}
		callId = CallId(callIdInt)

		// get the function name
		if !scanner.Scan() {
			return nil, fmt.Errorf("missing function name")
		}

		function = scanner.Text()
		if function == "" {
			return nil, fmt.Errorf("empty function name")
		}

	} else if req.verb == EMIT {
		// get event
		event = scanner.Text()
		if event == "" {
			return nil, fmt.Errorf("empty event name")
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
			return nil, fmt.Errorf("invalid format")
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
