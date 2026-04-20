import type { IncomingMessage, ServerResponse } from "http";
import {
  getStatus,
  getQueue,
  getAllResults,
  getResult,
  getAllPipelinesStatus,
  getPipelineStatus,
  getStageResults,
} from "./state.js";

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

  if (pathname === "/pipelines") {
    json(res, 200, getAllPipelinesStatus());
    return true;
  }

  const pipelineMatch = pathname.match(/^\/pipeline\/([^/]+)$/);
  if (pipelineMatch) {
    const name = decodeURIComponent(pipelineMatch[1]);
    const status = getPipelineStatus(name);
    if (status === null) {
      json(res, 404, { error: `no pipeline '${name}'` });
    } else {
      json(res, 200, status);
    }
    return true;
  }

  const stageResultsMatch = pathname.match(/^\/pipeline\/([^/]+)\/([^/]+)\/results$/);
  if (stageResultsMatch) {
    const pipeline = decodeURIComponent(stageResultsMatch[1]);
    const stage = decodeURIComponent(stageResultsMatch[2]);
    json(res, 200, getStageResults(pipeline, stage));
    return true;
  }

  return false;
}
