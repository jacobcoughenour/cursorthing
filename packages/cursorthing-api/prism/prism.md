
### PRISM
Presence Realtime Interaction Server Messaging

## CALL
Make function calls to the server in the following format:
```
CALL
<call_id>
<functionName>
[param format (json/text)]
[param as json or text]
```
and the server will respond with:
```
RES
<call_id>
[result format (json/text)]
[result json or text]
```
Or with an error:
```
ERR
[call_id]
<error message>
```

call ids are how the client can correspond the sent call message to the received response or error message. They must be a 64bit unsigned int (between 0 and 0xFFFFFFFF inclusive). They don't have to be unique but you should probably increment it for each call so save yourself the trouble.
If you have a malformed message that causes the server to not be able to parse the call_id, it will respond with an error message and a blank call_id.


## EMIT
Emits are like calls but without any response from the server. You won't get an acknowledgement or error.
```
EMIT
<event_name>
[format]
[data json or text]
```

## CAST
Broadcasts or casts are messages you receive from the server you didn't ask for. You will typically have a call that you need to make to subscribe to a topic to receive these messages.
```
CAST
<topic>
[format]
[data json or text]
```



