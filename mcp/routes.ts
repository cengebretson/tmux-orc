import type { IncomingMessage, ServerResponse } from "http";
import {
  getStatus,
  getQueue,
  getAllResults,
  getResult,
  getAllJobsStatus,
  getJobStatus,
  getStageResults,
} from "./state.js";

function json(res: ServerResponse, status: number, data: unknown): void {
  res.writeHead(status, { "Content-Type": "application/json" });
  res.end(JSON.stringify(data, null, 2));
}

type RouteHandler = (res: ServerResponse, ...params: string[]) => void;

const routes: Array<[RegExp, RouteHandler]> = [
  [/^\/status$/,  (res) => json(res, 200, getStatus())],
  [/^\/queue$/,   (res) => json(res, 200, getQueue())],
  [/^\/results$/, (res) => json(res, 200, getAllResults())],
  [/^\/jobs$/,    (res) => json(res, 200, getAllJobsStatus())],

  [/^\/result\/([^/]+)$/, (res, workerId) => {
    const result = getResult(workerId);
    result === null
      ? json(res, 404, { error: `no result for worker ${workerId}` })
      : json(res, 200, { worker_id: workerId, result });
  }],

  [/^\/job\/([^/]+)$/, (res, name) => {
    const status = getJobStatus(decodeURIComponent(name));
    status === null
      ? json(res, 404, { error: `no job '${name}'` })
      : json(res, 200, status);
  }],

  [/^\/job\/([^/]+)\/([^/]+)\/results$/, (res, job, stage) => {
    json(res, 200, getStageResults(decodeURIComponent(job), decodeURIComponent(stage)));
  }],
];

export function handleInspection(req: IncomingMessage, res: ServerResponse): boolean {
  if (req.method !== "GET") return false;

  const pathname = new URL(req.url!, "http://localhost").pathname;

  for (const [pattern, handler] of routes) {
    const match = pathname.match(pattern);
    if (match) {
      handler(res, ...match.slice(1));
      return true;
    }
  }

  return false;
}
