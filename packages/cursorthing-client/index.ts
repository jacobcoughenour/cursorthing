export function hello() {
	console.log("Hello from cursorthing-client!");

	new WebSocket("ws://localhost:8080");
}
