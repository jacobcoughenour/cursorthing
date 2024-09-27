import { CursorThingClient } from "cursorthing-client";

const client = new CursorThingClient("localhost");

client.on("connect", () => {
	console.log("Connected to server!");
});

export {};
