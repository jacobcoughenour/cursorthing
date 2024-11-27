import EventEmitter from "events";
import { Centrifuge, State } from "centrifuge";

/**
 * Represents a connection to a page on the server.
 * One instance should be created per tab.
 */
export class CursorThingPageConnection extends EventEmitter {
	private _centrifugeClient: Centrifuge;
	private _serverAddress: string;
	private _currentPagePath: string | null = null;

	constructor(server: string) {
		super();
		this._serverAddress = server;
		this._centrifugeClient = new Centrifuge([
			{
				transport: "websocket",
				endpoint: `ws://${this._serverAddress}/connection/websocket`,
			},
		]);
	}

	private async _ensureConnectionOrThrow() {
		if (this._serverAddress.trim().length === 0)
			throw new Error("No server address set");

		if (this._centrifugeClient.state === State.Disconnected) {
			this._centrifugeClient.connect();
		}
	}

	public async navigate(url: string) {
		await this._ensureConnectionOrThrow();

		const normalizedUrl = this._normalizeUrl(url);

		if (this._currentPagePath !== null) {
			// todo send navigate message

			console.log("navigating");
		} else {
			console.log("joining", normalizedUrl);
		}

		const sub = this._centrifugeClient.newSubscription(
			"page/" + normalizedUrl,
		);

		sub.on("publication", (ctx) => {
			console.log("publication", ctx);
		});

		sub.subscribe();

		// this._centrifugeClient.publish("join", { url });
	}

	private _normalizeUrl(url: string): string {
		var parsed = new URL(url);

		if (parsed.protocol !== "https") throw new Error("Invalid protocol");

		return parsed.hostname + parsed.pathname;
	}

	public async leave() {}
}
