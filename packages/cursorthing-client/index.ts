import EventEmitter from "events";
import { PrismClient } from "./prism";

export class CursorThingClient extends EventEmitter {
	private _prism: PrismClient;

	constructor(endpoint: string) {
		super();

		this._prism = new PrismClient(endpoint);
	}

	public async join(url: string) {
		const roomId = await this._prism.call("join", url);
		console.log(`Joined room ${roomId}`);
	}
}
