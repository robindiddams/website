let totalVisitors = 0;
let activeVisitors = 0;

const indexHTML = (config: {
	IP: string;
}) => `<!DOCTYPE html>
<html>  
  <head>  
    <script>
      const evtSource = new EventSource("/sse");
    
      evtSource.onmessage = function(e) {
        document.getElementById("active-visitors").textContent = e.data;
      };
    </script>
  </head>
  <body>
    <pre>Hello, ${config.IP}</pre>
    <pre>Thank you for visiting Robin's website!</pre>
    <pre>~ Robin</pre>
    <br/>
    <pre>Active visitors: <span id="active-visitors">0</span></pre>
    <pre>Total visitors: ${totalVisitors++}</pre>
  </body>
</html>
`;

// server.ts
const server = Bun.serve({
	port: 3000,
	routes: {
		"/": (req, srv) =>
			new Response(
				indexHTML({
					IP: srv.requestIP(req)?.address ?? "UNDADRESSED",
				}),
				{
					headers: {
						"Content-Type": "text/html",
					},
				},
			),
		"/sse": (req, srv) => {
			activeVisitors += 1;
			// let cancelStream: () => void;

			let interval: NodeJS.Timeout;

			const reader = new ReadableStream({
				start(controller) {
					let lastRead = activeVisitors;
					controller.enqueue(`data: ${activeVisitors}\n\n`);

					interval = setInterval(() => {
						if (activeVisitors !== lastRead) {
							controller.enqueue(`data: ${activeVisitors}\n\n`);
							lastRead = activeVisitors;
						}
					}, 300);
				},
				cancel() {
					clearInterval(interval);
					activeVisitors -= 1;
					console.log("Closing connection");
				},
			});

			return new Response(reader, {
				headers: {
					"Content-Type": "text/event-stream",
					"Cache-Control": "no-cache",
					Connection: "keep-alive",
				},
			});
		},
	},

	fetch(req) {
		return new Response("Not Found", { status: 404 });
	},
});

console.log(`Listening on http://localhost:${server.port}`);
