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
  stageDone,
  getStageResults,
  getJobStatus,
  getAllJobsStatus,
  resetJob,
} from "./state.js";

export const taskSchema = z.object({
  id: z.string(),
  role: z.string(),
  description: z.string(),
  job: z.string(),
  stage: z.string(),
  depends_on: z.array(z.string()).optional(),
});

export const mcp = new McpServer({
  name: "claude-agents-mcp",
  version: "0.1.0",
});

mcp.tool(
  "register_worker",
  "Worker registers itself with its tmux pane ID on startup",
  { worker_id: z.string(), pane_id: z.string() },
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
  { worker_id: z.string(), role: z.string() },
  async ({ worker_id, role }) => {
    const task = getTask(worker_id, role);
    return { content: [{ type: "text", text: task ? JSON.stringify(task) : "NO_TASKS" }] };
  }
);

mcp.tool(
  "submit_result",
  "Post completed output for a task (worker calls this when done)",
  { worker_id: z.string(), result: z.string() },
  async ({ worker_id, result }) => {
    submitResult(worker_id, result);
    return { content: [{ type: "text", text: "OK" }] };
  }
);

mcp.tool(
  "get_result",
  "Read a worker's submitted result (orchestrator calls this)",
  { worker_id: z.string() },
  async ({ worker_id }) => {
    const result = getResult(worker_id);
    return { content: [{ type: "text", text: result ?? "NO_RESULT" }] };
  }
);

mcp.tool(
  "all_done",
  "Returns true when the task queue is empty and all registered workers have submitted results",
  {},
  async () => {
    return { content: [{ type: "text", text: String(allDone()) }] };
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

mcp.tool(
  "stage_done",
  "Returns true when all tasks in a job stage have been submitted",
  { job: z.string(), stage: z.string() },
  async ({ job, stage }) => {
    return { content: [{ type: "text", text: String(stageDone(job, stage)) }] };
  }
);

mcp.tool(
  "get_stage_results",
  "Returns all worker results from a completed job stage",
  { job: z.string(), stage: z.string() },
  async ({ job, stage }) => {
    return { content: [{ type: "text", text: JSON.stringify(getStageResults(job, stage), null, 2) }] };
  }
);

mcp.tool(
  "get_jobs_status",
  "Returns stage breakdown for one job (if job is provided) or all active jobs",
  { job: z.string().optional() },
  async ({ job }) => {
    if (job) {
      const result = getJobStatus(job);
      if (result === null) {
        return { content: [{ type: "text", text: `No job '${job}' found` }] };
      }
      return { content: [{ type: "text", text: JSON.stringify(result, null, 2) }] };
    }
    return { content: [{ type: "text", text: JSON.stringify(getAllJobsStatus(), null, 2) }] };
  }
);

mcp.tool(
  "reset_job",
  "Clears all stage state for a job so the same pipeline can be rerun for a new feature in the same session",
  { job: z.string() },
  async ({ job }) => {
    const found = resetJob(job);
    return { content: [{ type: "text", text: found ? `Job '${job}' reset` : `No job '${job}' found` }] };
  }
);
