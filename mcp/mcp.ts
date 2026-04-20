import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import {
  loadTasks,
  registerWorker,
  getTask,
  submitResult,
  getResult,
  allDone,
  getStatus,
} from "./state.js";

export const taskSchema = z.object({
  id: z.string(),
  role: z.enum(["backend", "frontend", "code-review"]),
  description: z.string(),
  domain: z.string().optional(),
});

const roleSchema = z.enum(["backend", "frontend", "code-review"]);

export const mcp = new McpServer({
  name: "claude-agents-mcp",
  version: "0.1.0",
});

mcp.tool(
  "register_worker",
  "Worker registers itself with its tmux pane ID on startup",
  { worker_id: z.number(), pane_id: z.string() },
  async ({ worker_id, pane_id }) => {
    registerWorker(worker_id, pane_id);
    return { content: [{ type: "text", text: "OK" }] };
  }
);

mcp.tool(
  "load_tasks",
  "Load the task queue (orchestrator calls this on startup)",
  { tasks: z.array(taskSchema) },
  async ({ tasks }) => {
    const count = loadTasks(tasks);
    return { content: [{ type: "text", text: `Loaded ${count} tasks` }] };
  }
);

mcp.tool(
  "get_task",
  "Pull the next role-matched task from the queue (worker calls this when ready)",
  { worker_id: z.number(), role: roleSchema },
  async ({ worker_id, role }) => {
    const task = getTask(worker_id, role);
    return { content: [{ type: "text", text: task ? JSON.stringify(task) : "NO_TASKS" }] };
  }
);

mcp.tool(
  "submit_result",
  "Post completed output for a task (worker calls this when done)",
  { worker_id: z.number(), result: z.string() },
  async ({ worker_id, result }) => {
    submitResult(worker_id, result);
    return { content: [{ type: "text", text: "OK" }] };
  }
);

mcp.tool(
  "get_result",
  "Read a worker's submitted result (orchestrator calls this)",
  { worker_id: z.number() },
  async ({ worker_id }) => {
    const result = getResult(worker_id);
    return { content: [{ type: "text", text: result ?? "NO_RESULT" }] };
  }
);

mcp.tool(
  "all_done",
  "Returns true when the task queue is empty and all workers have submitted results",
  { worker_count: z.number() },
  async ({ worker_count }) => {
    return { content: [{ type: "text", text: String(allDone(worker_count)) }] };
  }
);

mcp.tool(
  "get_status",
  "Returns queue depth and each known worker's status (working | submitted)",
  {},
  async () => {
    return { content: [{ type: "text", text: JSON.stringify(getStatus(), null, 2) }] };
  }
);
