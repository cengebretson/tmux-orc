import { SSEServerTransport } from "@modelcontextprotocol/sdk/server/sse.js";
import { createServer, type IncomingMessage, type ServerResponse } from "http";
import { mcp } from "./mcp.js";
import { handleInspection } from "./routes.js";

const transports = new Map<string, SSEServerTransport>();

const portArg = process.argv.indexOf("--port");
const port = portArg !== -1 ? parseInt(process.argv[portArg + 1]) : 7777;

if (!process.env.PROJECT_DIR) {
  console.warn("WARNING: PROJECT_DIR is not set — knowledge files will be written relative to cwd. Start the server via 'cli.ts start-mcp' to fix this.");
}

const httpServer = createServer(
  async (req: IncomingMessage, res: ServerResponse) => {
    if (handleInspection(req, res)) return;

    const url = new URL(req.url!, `http://localhost:${port}`);

    if (req.method === "GET" && url.pathname === "/sse") {
      const transport = new SSEServerTransport("/messages", res);
      transports.set(transport.sessionId, transport);
      res.on("close", () => transports.delete(transport.sessionId));
      await mcp.connect(transport);
      return;
    }

    if (req.method === "POST" && url.pathname === "/messages") {
      const sessionId = url.searchParams.get("sessionId")!;
      const transport = transports.get(sessionId);
      if (!transport) {
        res.writeHead(404).end();
        return;
      }
      await transport.handlePostMessage(req, res);
      return;
    }

    res.writeHead(404).end();
  }
);

httpServer.listen(port, () => {
  console.log(`claude-agents-mcp listening on http://localhost:${port}`);
});
