export interface Task {
  id: string;
  role: string;
  description: string;
  job: string;
  stage: string;
  depends_on?: string[];
}

export type StageStatus = "active" | "complete";

export interface StageInfo {
  status: StageStatus;
  taskCount: number;
  resultCount: number;
  results: Record<string, string>;
}

export interface JobStatus {
  stages: Record<string, StageInfo>;
}

export interface Status {
  queue: number;
  allDone: boolean;
  workers: Record<string, {
    status: "idle" | "working" | "submitted" | "blocked";
    paneId?: string;
    currentTask?: Task;
    blockedReason?: string;
    lastActivityAt?: number;
  }>;
}
