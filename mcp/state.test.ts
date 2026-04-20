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
const review: Task = { id: "3", role: "review", description: "Review auth PR" };

beforeEach(() => reset());

describe("registerWorker", () => {
  it("stores the pane ID for a worker", () => {
    registerWorker("bob", "%12");
    expect(getStatus().workers["bob"].paneId).toBe("%12");
  });

  it("preserves existing worker state when registering", () => {
    loadTasks([frontend]);
    getTask("bob", "frontend");
    registerWorker("bob", "%12");
    const w = getStatus().workers["bob"];
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
    expect(getTask("bob", "frontend")).toEqual(frontend);
    expect(getTask("alice", "backend")).toEqual(backend);
  });
});

describe("getTask", () => {
  it("returns a task matching the worker's role", () => {
    loadTasks([backend, frontend]);
    expect(getTask("bob", "frontend")).toEqual(frontend);
  });

  it("skips tasks that don't match the role", () => {
    loadTasks([backend, backend, frontend]);
    expect(getTask("bob", "frontend")).toEqual(frontend);
    expect(getTask("alice", "backend")).toEqual(backend);
  });

  it("returns null when no matching task exists", () => {
    loadTasks([backend]);
    expect(getTask("bob", "frontend")).toBeNull();
  });

  it("returns null when queue is empty", () => {
    expect(getTask("bob", "backend")).toBeNull();
  });

  it("marks the worker as working with the claimed task", () => {
    loadTasks([frontend]);
    getTask("bob", "frontend");
    const status = getStatus();
    expect(status.workers["bob"].status).toBe("working");
    expect(status.workers["bob"].currentTask).toEqual(frontend);
  });

  it("does not mark worker as working when no matching task", () => {
    getTask("bob", "backend");
    expect(getStatus().workers["bob"]).toBeUndefined();
  });
});

describe("submitResult / getResult", () => {
  it("stores and retrieves a result by worker id", () => {
    submitResult("bob", "done");
    expect(getResult("bob")).toBe("done");
  });

  it("returns null for a worker that hasn't submitted", () => {
    expect(getResult("nobody")).toBeNull();
  });

  it("overwrites a previous result from the same worker", () => {
    submitResult("bob", "v1");
    submitResult("bob", "v2");
    expect(getResult("bob")).toBe("v2");
  });

  it("marks the worker as submitted", () => {
    submitResult("bob", "done");
    expect(getStatus().workers["bob"].status).toBe("submitted");
  });
});

describe("allDone", () => {
  it("returns false when tasks remain", () => {
    loadTasks([frontend]);
    expect(allDone(2)).toBe(false);
  });

  it("returns false when queue is empty but not all workers submitted", () => {
    submitResult("bob", "done");
    expect(allDone(2)).toBe(false);
  });

  it("returns true when queue is empty and all workers submitted", () => {
    submitResult("bob", "done");
    submitResult("alice", "done");
    expect(allDone(2)).toBe(true);
  });

  it("returns false mid-run with tasks and partial results", () => {
    loadTasks([frontend, backend]);
    getTask("bob", "frontend");
    submitResult("bob", "done");
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
    getTask("bob", "frontend");
    getTask("alice", "backend");
    submitResult("bob", "done");
    const status = getStatus();
    expect(status.queue).toBe(0);
    expect(status.workers["bob"].status).toBe("submitted");
    expect(status.workers["alice"].status).toBe("working");
    expect(status.workers["alice"].currentTask).toEqual(backend);
  });

  it("updates worker from working to submitted", () => {
    loadTasks([frontend]);
    getTask("bob", "frontend");
    expect(getStatus().workers["bob"].status).toBe("working");
    submitResult("bob", "done");
    expect(getStatus().workers["bob"].status).toBe("submitted");
  });
});
