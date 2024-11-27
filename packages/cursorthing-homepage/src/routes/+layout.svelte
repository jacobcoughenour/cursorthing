<script>
	import "../app.css";
	import { browser } from "$app/environment";
	import { beforeNavigate, afterNavigate } from "$app/navigation";
	import { CursorThingPageConnection } from "cursorthing-client";

	let { children } = $props();

	if (browser) {
		const connection = new CursorThingPageConnection("localhost:8080");

		/**
		 * @param url {URL}
		 */
		const navigate = (url) => {
			if (url.host === "localhost:5173") {
				// pretend we're on the prod domain
				url.host = "cursorthing.com";
			}
			connection.navigate(url.toString());
		};

		beforeNavigate((e) => {
			if (e.type === "leave") {
				connection.leave();
			} else if (!!e.to?.url) {
				navigate(e.to?.url);
			}
		});

		afterNavigate((e) => {
			if (!!e.to?.url) navigate(e.to.url);
		});
	}
</script>

<nav>
	<a href="/">Home</a>
	<a href="/about">About</a>
</nav>

{@render children()}
