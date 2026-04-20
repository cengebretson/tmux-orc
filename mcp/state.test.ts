import { describe, it, expect, beforeEach } from "bun:test";
import {
  loadTasks,
  registerWorker,
  getTask,
  submitResult,
  getResult,
  allDone,
  getStatus,
  reset,
  type Task,
} from "./state.js";

const frontend: Task = { id: "1", role: "frontend", description: "Build login form", domain: "src/frontend/" };
const backend: Task = { id: "2", role: "backend", description: "Build login endpoint", domain: "src/backend/" };
const review: Task = { id: "3", role: "code-review", description: "Review auth PR" };

beforeEach(() => reset());

describe("registerWorker", () => {
  it("stores the pane ID for a worker", () => {
    registerWorker(2, "%12");
    expect(getStatus().workers[2].paneId).toBe("%12");
  });

  it("preserves existing worker state when registering", () => {
    loadTasks([frontend]);
    getTask(2, "frontend");
    registerWorker(2, "%12");
    const w = getStatus().workers[2];
    expect(w.paneId).toBe("%12");
    expect(w.status).toBe("working");
  });
});

describe("loadTasks", () => {
  it("adds tasks to the queue and returns count", () => {
    expect(loadTasks([frontend, backend, review])).toBe(3);
  });

  it("appends to existing tasks", () => {
    loadTasks([frontend]);
    loadTasks([backend]);
    expect(getTask(2, "frontend")).toEqual(frontend);
    expect(getTask(3, "backend")).toEqual(backend);
  });
});

describe("getTask", () => {
  it("returns a task matching the worker's role", () => {
    loadTasks([backend, frontend]);
    expect(getTask(2, "frontend")).toEqual(frontend);
  });

  it("skips tasks that don't match the role", () => {
    loadTasks([backend, backend, frontend]);
    expect(getTask(2, "frontend")).toEqual(frontend);
    expect(getTask(3, "backend")).toEqual(backend);
  });

  it("returns null when no matching task exists", () => {
    loadTasks([backend]);
    expect(getTask(2, "frontend")).toBeNull();
  });

  it("returns null when queue is empty", () => {
    expect(getTask(2, "backend")).toBeNull();
  });

  it("marks the worker as working with the claimed task", () => {
    loadTasks([frontend]);
    getTask(2, "frontend");
    const status = getStatus();
    expect(status.workers[2].status).toBe("working");
    expect(status.workers[2].currentTask).toEqual(frontend);
  });

  it("does not mark worker as working when no matching task", () => {
    getTask(2, "backend");
    expect(getStatus().workers[2]).toBeUndefined();
  });
});

describe("submitResult / getResult", () => {
  it("stores and retrieves a result by worker id", () => {
    submitResult(2, "done");
    expect(getResult(2)).toBe("done");
  });

  it("returns null for a worker that hasn't submitted", () => {
    expect(getResult(99)).toBeNull();
  });

  it("overwrites a previous result from the same worker", () => {
    submitResult(2, "v1");
    submitResult(2, "v2");
    expect(getResult(2)).toBe("v2");
  });

  it("marks the worker as submitted", () => {
    submitResult(2, "done");
    expect(getStatus().workers[2].status).toBe("submitted");
  });
});

describe("allDone", () => {
  it("returns false when tasks remain", () => {
    loadTasks([frontend]);
    expect(allDone(2)).toBe(false);
  });

  it("returns false when queue is empty but not all workers submitted", () => {
    submitResult(2, "done");
    expect(allDone(2)).toBe(false);
  });

  it("returns true when queue is empty and all workers submitted", () => {
    submitResult(2, "done");
    submitResult(3, "done");
    expect(allDone(2)).toBe(true);
  });

  it("returns false mid-run with tasks and partial results", () => {
    loadTasks([frontend, backend]);
    getTask(2, "frontend");
    submitResult(2, "done");
    expect(allDone(2)).toBe(false);
  });
});

describe("getStatus", () => {
  it("returns empty status initially", () => {
    loadTasks([frontend]);
    expect(getStatus()).toEqual({ queue: 1, workers: {} });
  });

  it("reflects working and submitted states together", () => {
    loadTasks([frontend, backend]);
    getTask(2, "frontend");
    getTask(3, "backend");
    submitResult(2, "done");
    const status = getStatus();
    expect(status.queue).toBe(0);
    expect(status.workers[2].status).toBe("submitted");
    expect(status.workers[3].status).toBe("working");
    expect(status.workers[3].currentTask).toEqual(backend);
  });

  it("updates worker from working to submitted", () => {
    loadTasks([frontend]);
    getTask(2, "frontend");
    expect(getStatus().workers[2].status).toBe("working");
    submitResult(2, "done");
    expect(getStatus().workers[2].status).toBe("submitted");
  });
});
