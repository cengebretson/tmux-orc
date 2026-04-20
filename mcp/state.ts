import { appendFileSync, mkdirSync } from "fs";
import type { Task, StageStatus, StageInfo, JobStatus, Status } from "./types.js";
export type { Task, StageStatus, StageInfo, JobStatus, Status };

interface WorkerState {
  status: "idle" | "working" | "submitted" | "blocked";
  paneId?: string;
  currentTask?: Task;
  blockedReason?: string;
}

interface StageState {
  taskCount: number;
  results: Map<string, string>;
}

const taskQueue: Task[] = [];
const results = new Map<string, string>();
const workerState = new Map<string, WorkerState>();
const jobState = new Map<string, Map<string, StageState>>();

function getOrCreateStage(job: string, stage: string): StageState {
  if (!jobState.has(job)) jobState.set(job, new Map());
  const stages = jobState.get(job)!;
  if (!stages.has(stage)) stages.set(stage, { taskCount: 0, results: new Map() });
  return stages.get(stage)!;
}

function stageStatus(s: StageState): StageStatus {
  return s.results.size >= s.taskCount ? "complete" : "active";
}

export function loadTasks(tasks: Task[]): { count: number; error?: string } {
  const newJobs = new Set(tasks.map((t) => t.job));
  for (const job of newJobs) {
    if (jobState.has(job)) {
      return { count: 0, error: `job '${job}' already exists — use reset_job to rerun it` };
    }
  }
  for (const task of tasks) {
    taskQueue.push(task);
    getOrCreateStage(task.job, task.stage).taskCount++;
  }
  return { count: tasks.length };
}

export function registerWorker(workerId: string, paneId: string): void {
  const existing = workerState.get(workerId);
  workerState.set(workerId, { status: "idle", ...existing, paneId });
}

export function getTask(workerId: string, role: string): Task | null {
  const idx = taskQueue.findIndex(
    (t) => t.role === role &&
      (t.depends_on ?? []).every((stage) => stageDone(t.job, stage))
  );
  if (idx === -1) return null;
  const [task] = taskQueue.splice(idx, 1);
  const existing = workerState.get(workerId);
  workerState.set(workerId, { ...existing, status: "working", currentTask: task });
  return task;
}

export function submitResult(workerId: string, result: string): void {
  results.set(workerId, result);
  const existing = workerState.get(workerId);
  workerState.set(workerId, { ...existing, status: "submitted" });

  const task = existing?.currentTask;
  if (task) {
    getOrCreateStage(task.job, task.stage).results.set(workerId, result);
  }
}

export function getResult(workerId: string): string | null {
  return results.get(workerId) ?? null;
}

export function stageDone(job: string, stage: string): boolean {
  const s = jobState.get(job)?.get(stage);
  if (!s) return false;
  return s.results.size >= s.taskCount;
}

export function getStageResults(job: string, stage: string): Record<string, string> {
  const s = jobState.get(job)?.get(stage);
  return s ? Object.fromEntries(s.results) : {};
}

export function getJobStatus(job: string): JobStatus | null {
  const stages = jobState.get(job);
  if (!stages) return null;
  const out: Record<string, StageInfo> = {};
  for (const [name, s] of stages) {
    out[name] = {
      status: stageStatus(s),
      taskCount: s.taskCount,
      resultCount: s.results.size,
      results: Object.fromEntries(s.results),
    };
  }
  return { stages: out };
}

export function getAllJobsStatus(): Record<string, JobStatus> {
  const out: Record<string, JobStatus> = {};
  for (const [name] of jobState) {
    out[name] = getJobStatus(name)!;
  }
  return out;
}

export function allDone(): boolean {
  if (workerState.size === 0) return false;
  return taskQueue.length === 0 &&
    Array.from(workerState.values()).every(w => w.status === "submitted" || w.status === "idle");
}


export function getStatus(): Status {
  return {
    queue: taskQueue.length,
    workers: Object.fromEntries(workerState),
  };
}

export function getQueue(): Task[] {
  return [...taskQueue];
}

export function getAllResults(): Record<string, string> {
  return Object.fromEntries(results);
}

export function resetJob(job: string): boolean {
  if (!jobState.has(job)) return false;
  jobState.delete(job);
  const before = taskQueue.length;
  taskQueue.splice(0, taskQueue.length, ...taskQueue.filter((t) => t.job !== job));
  return true;
}

export function reportBlocked(workerId: string, reason: string): void {
  const existing = workerState.get(workerId);
  workerState.set(workerId, { ...existing, status: "blocked", blockedReason: reason });
}

export function resolveBlock(workerId: string, resolution: string): void {
  const existing = workerState.get(workerId);
  if (!existing) return;
  workerState.set(workerId, { ...existing, status: "working", blockedReason: undefined });

  const task = existing.currentTask;
  if (task) {
    const dir = `.claude/knowledge`;
    const date = new Date().toISOString().slice(0, 10);
    const entry = [
      ``,
      `## ${date} | job: ${task.job} | stage: ${task.stage}`,
      `**Blocked:** ${existing.blockedReason ?? "(no reason given)"}`,
      `**Resolution:** ${resolution}`,
      ``,
    ].join("\n");
    mkdirSync(dir, { recursive: true });
    appendFileSync(`${dir}/${task.role}.md`, entry);
  }
}

export function reset(): void {
  taskQueue.length = 0;
  results.clear();
  workerState.clear();
  jobState.clear();
}
