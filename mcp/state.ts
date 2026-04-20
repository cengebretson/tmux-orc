export type TaskRole = "backend" | "frontend" | "code-review";

export interface Task {
  id: string;
  role: TaskRole;
  description: string;
  domain?: string;
}

interface WorkerState {
  status: "working" | "submitted";
  paneId?: string;
  currentTask?: Task;
}

const taskQueue: Task[] = [];
const results = new Map<number, string>();
const workerState = new Map<number, WorkerState>();

export function loadTasks(tasks: Task[]): number {
  taskQueue.push(...tasks);
  return tasks.length;
}

export function registerWorker(workerId: number, paneId: string): void {
  const existing = workerState.get(workerId);
  workerState.set(workerId, { ...existing, paneId });
}

export function getTask(workerId: number, role: TaskRole): Task | null {
  const idx = taskQueue.findIndex((t) => t.role === role);
  if (idx === -1) return null;
  const [task] = taskQueue.splice(idx, 1);
  const existing = workerState.get(workerId);
  workerState.set(workerId, { ...existing, status: "working", currentTask: task });
  return task;
}

export function submitResult(workerId: number, result: string): void {
  results.set(workerId, result);
  const existing = workerState.get(workerId);
  workerState.set(workerId, { ...existing, status: "submitted" });
}

export function getResult(workerId: number): string | null {
  return results.get(workerId) ?? null;
}

export function allDone(workerCount: number): boolean {
  return taskQueue.length === 0 && results.size >= workerCount;
}

export interface Status {
  queue: number;
  workers: Record<number, WorkerState>;
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

export function getAllResults(): Record<number, string> {
  return Object.fromEntries(results);
}

export function reset(): void {
  taskQueue.length = 0;
  results.clear();
  workerState.clear();
}
