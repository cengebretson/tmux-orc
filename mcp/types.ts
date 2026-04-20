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
  workers: Record<string, {
    status: "idle" | "working" | "submitted";
    paneId?: string;
    currentTask?: Task;
  }>;
}
