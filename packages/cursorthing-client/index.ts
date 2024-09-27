import EventEmitter from "events";

export class CursorThingClient extends EventEmitter {
	private _ws: WebSocket;

	constructor(endpoint: string) {
		super();

		// todo only establish connection when content script when a group is joined
		this._ws = new WebSocket(`ws://${endpoint}:8080`);

		this._ws.onopen = () => {
			this.emit("connect");
		};
		this._ws.onmessage = (event) => {
			this.emit("message", JSON.parse(event.data));
		};
		this._ws.onclose = () => {
			this.emit("disconnect");
		};
		this._ws.onerror = (error) => {
			this.emit("error", error);
		};
	}
}
