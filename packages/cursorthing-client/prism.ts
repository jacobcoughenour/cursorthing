import EventEmitter from "events";

const DEBUG = true;

export class PrismClient {
	private _ws: WebSocket;
	private _ee: EventEmitter;

	private _endpoint: string;
	private _connectionPromise: Promise<void> | undefined;

	constructor(endpoint: string) {
		this._ee = new EventEmitter();
		this._endpoint = endpoint;
	}

	private async ensureConnected() {
		if (this._connectionPromise) return this._connectionPromise;
		this._connectionPromise = new Promise((resolve, reject) => {
			this._ws = new WebSocket(`ws://${this._endpoint}:8080/ws`);

			this._ws.onopen = () => {
				this.onConnect();
				resolve(undefined);
			};
			this._ws.onmessage = (event) => {
				this.onMessage(event);
			};
			this._ws.onclose = () => {
				this.onDisconnect();
				reject("Connection closed");
				this._connectionPromise = undefined;
			};
			this._ws.onerror = (error) => {
				this.onError(error);
				reject(error);
				this._connectionPromise = undefined;
			};
		});
		return this._connectionPromise;
	}

	public on(event: string, listener: (...args: any[]) => void) {
		this._ee.on(event, listener);
	}

	private onConnect() {
		console.log("Connected to server");
		this._ee.emit("connect");
	}

	private onDisconnect() {
		console.log("Disconnected from server");
		this._ee.emit("disconnect");
	}

	private onError(error: Event) {
		console.error(error);
		this._ee.emit("error", error);
	}

	private _lastCallId = 0;
	private nextCallId(): number {
		this._lastCallId++;
		if (this._lastCallId >= 0xffffffff) this._lastCallId = 0;
		return this._lastCallId;
	}

	private _pendingCalls: Map<
		number,
		[
			// resolve
			(value: any) => void,
			// reject
			(reason: string | object | undefined) => void,
		]
	> = new Map();

	public async call<T>(method: string, params?: string | object): Promise<T> {
		await this.ensureConnected();

		const callId = this.nextCallId();
		const header = `CALL\n${callId}\n${method}`;

		return new Promise((resolve, reject) => {
			// make sure we register the call before sending the request
			this._pendingCalls.set(callId, [resolve, reject]);

			let message: string;
			if (typeof params === "string") {
				message = `${header}\nTEXT\n${params}`;
			} else if (typeof params === "object") {
				message = `${header}\nJSON\n${JSON.stringify(params)}`;
			} else {
				message = `${header}`;
			}

			console.log("Sending message", message);

			this._ws.send(message);
		});
	}

	private onMessage(event: MessageEvent<any>) {
		if (DEBUG) console.log("Received message", event.data);

		const lines = event.data.split("\n");

		const verb = lines[0];

		if (verb === "RES") {
			const callId = parseInt(lines[1]);
			if (isNaN(callId)) {
				console.error("Invalid callId", lines[1]);
				return;
			}

			const prom = this._pendingCalls.get(callId);
			if (!prom) {
				console.error(
					"No pending call for response with callId",
					callId,
				);
				return;
			}
			const [resolve, reject] = prom;

			const dataType = lines[2];
			const body = lines.slice(3).join("\n");

			if (dataType === "TEXT") {
				resolve(body);
			} else if (dataType === "JSON") {
				try {
					resolve(JSON.parse(body));
				} catch (e) {
					console.error("The server sent invalid JSON", body);
					reject(e);
				}
			} else {
				reject("Invalid response type");
			}

			this._pendingCalls.delete(callId);
		} else if (verb === "CAST") {
			// todo
		}
	}
}
