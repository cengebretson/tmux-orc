export type TaskRole = string;

export interface Task {
  id: string;
  role: TaskRole;
  description: string;
  domain?: string | string[];
  pipeline?: string;
  stage?: string;
}

interface WorkerState {
  status: "working" | "submitted";
  paneId?: string;
  currentTask?: Task;
}

interface StageState {
  taskCount: number;
  results: Map<string, string>; // workerId -> result
}

export type StageStatus = "pending" | "active" | "complete";

export interface StageInfo {
  status: StageStatus;
  taskCount: number;
  resultCount: number;
  results: Record<string, string>;
}

export interface PipelineStatus {
  stages: Record<string, StageInfo>;
}

const taskQueue: Task[] = [];
const results = new Map<string, string>();
const workerState = new Map<string, WorkerState>();
const pipelineState = new Map<string, Map<string, StageState>>();

function getOrCreateStage(pipeline: string, stage: string): StageState {
  if (!pipelineState.has(pipeline)) pipelineState.set(pipeline, new Map());
  const stages = pipelineState.get(pipeline)!;
  if (!stages.has(stage)) stages.set(stage, { taskCount: 0, results: new Map() });
  return stages.get(stage)!;
}

function stageStatus(s: StageState): StageStatus {
  if (s.taskCount === 0) return "pending";
  if (s.results.size >= s.taskCount) return "complete";
  return "active";
}

export function loadTasks(tasks: Task[]): number {
  for (const task of tasks) {
    taskQueue.push(task);
    if (task.pipeline && task.stage) {
      const s = getOrCreateStage(task.pipeline, task.stage);
      s.taskCount++;
    }
  }
  return tasks.length;
}

export function registerWorker(workerId: string, paneId: string): void {
  const existing = workerState.get(workerId);
  workerState.set(workerId, { ...existing, paneId });
}

export function getTask(workerId: string, role: TaskRole): Task | null {
  const idx = taskQueue.findIndex((t) => t.role === role);
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

  // attribute to pipeline/stage via the worker's current task
  const task = existing?.currentTask;
  if (task?.pipeline && task?.stage) {
    const s = getOrCreateStage(task.pipeline, task.stage);
    s.results.set(workerId, result);
  }
}

export function getResult(workerId: string): string | null {
  return results.get(workerId) ?? null;
}

export function stageDone(pipeline: string, stage: string): boolean {
  const s = pipelineState.get(pipeline)?.get(stage);
  if (!s || s.taskCount === 0) return false;
  return s.results.size >= s.taskCount;
}

export function getStageResults(pipeline: string, stage: string): Record<string, string> {
  const s = pipelineState.get(pipeline)?.get(stage);
  return s ? Object.fromEntries(s.results) : {};
}

export function getPipelineStatus(pipeline: string): PipelineStatus | null {
  const stages = pipelineState.get(pipeline);
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

export function getAllPipelinesStatus(): Record<string, PipelineStatus> {
  const out: Record<string, PipelineStatus> = {};
  for (const [name] of pipelineState) {
    out[name] = getPipelineStatus(name)!;
  }
  return out;
}

export function allDone(workerCount: number): boolean {
  return taskQueue.length === 0 && results.size >= workerCount;
}

export interface Status {
  queue: number;
  workers: Record<string, WorkerState>;
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

export function reset(): void {
  taskQueue.length = 0;
  results.clear();
  workerState.clear();
  pipelineState.clear();
}
