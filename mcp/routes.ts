import type { IncomingMessage, ServerResponse } from "http";
import { getStatus, getQueue, getAllResults, getResult } from "./state.js";

function json(res: ServerResponse, status: number, data: unknown): void {
  res.writeHead(status, { "Content-Type": "application/json" });
  res.end(JSON.stringify(data, null, 2));
}

export function handleInspection(
  req: IncomingMessage,
  res: ServerResponse
): boolean {
  if (req.method !== "GET") return false;

  const pathname = new URL(req.url!, "http://localhost").pathname;

  if (pathname === "/status") {
    json(res, 200, getStatus());
    return true;
  }

  if (pathname === "/queue") {
    json(res, 200, getQueue());
    return true;
  }

  if (pathname === "/results") {
    json(res, 200, getAllResults());
    return true;
  }

  const resultMatch = pathname.match(/^\/result\/([^/]+)$/);
  if (resultMatch) {
    const workerId = resultMatch[1];
    const result = getResult(workerId);
    if (result === null) {
      json(res, 404, { error: `no result for worker ${workerId}` });
    } else {
      json(res, 200, { worker_id: workerId, result });
    }
    return true;
  }

  return false;
}
