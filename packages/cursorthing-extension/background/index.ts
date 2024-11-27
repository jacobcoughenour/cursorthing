import { CursorThingClient } from "cursorthing-client";

const client = new CursorThingClient("localhost:8080");

// todo should debounce navigation events to avoid spamming the server when
// the client is getting redirected around

client.join("https://google.com");

export {};
