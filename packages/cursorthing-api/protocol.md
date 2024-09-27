

Make function calls to the server in the following format:
```
CALL
<call_id>
<functionName>
[format]
[args as json or text]
```
and the server will respond with:
```
RES
<call_id>
[format (json/text)]
[result json or text]
```
Or with an er ror:
```
ERR
<call_id>
<error code>
<error message>
```

emits are like calls but without any response from the server.
```
EMIT
<event_name>
[format]
[data json or text]
```

then broadcasts are messages you get from the server you didn't ask for.
```
CAST
<topic>
[format]
[data json or text]
```



