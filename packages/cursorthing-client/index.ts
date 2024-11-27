import EventEmitter from "events";
import { Centrifuge } from "centrifuge";

export class CursorThingClient extends EventEmitter {
	private _centrifugeClient: Centrifuge;

	constructor(server: string) {
		super();

		this._centrifugeClient = new Centrifuge([
			{
				transport: "websocket",
				endpoint: `ws://${server}/connection/websocket`,
			},
		]);

		const sub = this._centrifugeClient.newSubscription("test");

		sub.on("publication", (ctx) => {
			console.log("publication", ctx);
		});

		sub.subscribe();

		this._centrifugeClient.connect();
	}

	public async join(url: string) {}
}
