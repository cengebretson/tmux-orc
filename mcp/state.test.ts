import { describe, it, expect, beforeEach } from "bun:test";
import { mkdtempSync, readFileSync, rmSync } from "fs";
import { tmpdir } from "os";
import { join } from "path";
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
  reportBlocked,
  resolveBlock,
  reset,
  type Task,
} from "./state.js";

const frontend: Task = { id: "1", role: "frontend", description: "Build login form",     job: "auth-login", stage: "build" };
const backend: Task  = { id: "2", role: "backend",  description: "Build login endpoint", job: "auth-login", stage: "build" };
const review: Task   = { id: "3", role: "review",   description: "Review auth PR",       job: "auth-login", stage: "review" };

const pFrontend: Task = { ...frontend, id: "p1" };
const pReview: Task   = { ...review,   id: "p2" };
const pSecurity: Task = { id: "p3", role: "security", description: "Security check", job: "auth-login", stage: "security" };
const pGit: Task      = { id: "p4", role: "git",      description: "Create PR",      job: "auth-login", stage: "ship" };

beforeEach(() => reset());

describe("registerWorker", () => {
  it("stores the pane ID and sets status to idle", () => {
    registerWorker("bob", "%12");
    const w = getStatus().workers["bob"];
    expect(w.paneId).toBe("%12");
    expect(w.status).toBe("idle");
  });

  it("preserves existing worker state when re-registering", () => {
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
    expect(loadTasks([frontend, backend, review]).count).toBe(3);
  });

  it("appends tasks for different jobs", () => {
    const t2: Task = { id: "x1", role: "backend", description: "API work", job: "dashboard", stage: "build" };
    loadTasks([frontend]);
    loadTasks([t2]);
    expect(getTask("bob", "frontend")).toEqual(frontend);
    expect(getTask("alice", "backend")).toEqual(t2);
  });

  it("registers job stage task counts", () => {
    loadTasks([pFrontend, pReview]);
    const status = getJobStatus("auth-login")!;
    expect(status.stages["build"].taskCount).toBe(1);
    expect(status.stages["review"].taskCount).toBe(1);
  });

  it("rejects tasks for a job that already exists", () => {
    loadTasks([pFrontend]);
    const result = loadTasks([pReview]);
    expect(result.error).toMatch(/auth-login/);
    expect(result.count).toBe(0);
  });

  it("does not partially load tasks when a job conflict is detected", () => {
    loadTasks([pFrontend]);
    const t2: Task = { id: "x1", role: "backend", description: "API", job: "dashboard", stage: "build" };
    const result = loadTasks([t2, pReview]);
    expect(result.error).toMatch(/auth-login/);
    expect(getTask("alice", "backend")).toBeNull();
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

  it("attributes result to job stage via current task", () => {
    loadTasks([pFrontend]);
    getTask("bob", "frontend");
    submitResult("bob", "login form done");
    expect(getStageResults("auth-login", "build")).toEqual({ bob: "login form done" });
  });
});

describe("allDone", () => {
  it("returns false when no workers have registered", () => {
    expect(allDone()).toBe(false);
  });

  it("returns false when tasks remain in the queue", () => {
    loadTasks([pFrontend]);
    expect(allDone()).toBe(false);
  });

  it("returns false when some workers are still working", () => {
    loadTasks([pFrontend, pReview]);
    getTask("bob", "frontend");
    getTask("rex", "review");
    submitResult("bob", "done");
    expect(allDone()).toBe(false);
  });

  it("returns true when queue is empty and all registered workers have submitted", () => {
    loadTasks([pFrontend, pReview]);
    getTask("bob", "frontend");
    getTask("rex", "review");
    submitResult("bob", "done");
    submitResult("rex", "done");
    expect(allDone()).toBe(true);
  });
});

describe("getStatus", () => {
  it("returns empty status initially", () => {
    loadTasks([pFrontend]);
    expect(getStatus()).toEqual({ queue: 1, workers: {} });
  });

  it("reflects working and submitted states together", () => {
    loadTasks([pFrontend, pReview]);
    getTask("bob", "frontend");
    getTask("rex", "review");
    submitResult("bob", "done");
    const status = getStatus();
    expect(status.queue).toBe(0);
    expect(status.workers["bob"].status).toBe("submitted");
    expect(status.workers["rex"].status).toBe("working");
    expect(status.workers["rex"].currentTask).toEqual(pReview);
  });

  it("updates worker from working to submitted", () => {
    loadTasks([pFrontend]);
    getTask("bob", "frontend");
    expect(getStatus().workers["bob"].status).toBe("working");
    submitResult("bob", "done");
    expect(getStatus().workers["bob"].status).toBe("submitted");
  });
});

describe("depends_on", () => {
  it("withholds a task until its dependency stage is complete", () => {
    const build:  Task = { id: "p1", role: "frontend", description: "Build",  job: "auth-login", stage: "build" };
    const review: Task = { id: "p2", role: "review",   description: "Review", job: "auth-login", stage: "review", depends_on: ["build"] };
    loadTasks([build, review]);

    expect(getTask("rex", "review")).toBeNull();

    getTask("bob", "frontend");
    submitResult("bob", "done");

    expect(getTask("rex", "review")).toEqual(review);
  });

  it("withholds a task until all dependency stages are complete", () => {
    const build:    Task = { id: "p1", role: "frontend", description: "Build",    job: "auth-login", stage: "build" };
    const review:   Task = { id: "p2", role: "review",   description: "Review",   job: "auth-login", stage: "review",   depends_on: ["build"] };
    const security: Task = { id: "p3", role: "security", description: "Security", job: "auth-login", stage: "security", depends_on: ["build"] };
    const ship:     Task = { id: "p4", role: "git",      description: "Ship",     job: "auth-login", stage: "ship",     depends_on: ["review", "security"] };
    loadTasks([build, review, security, ship]);

    getTask("bob", "frontend");
    submitResult("bob", "build done");

    getTask("rex", "review");
    expect(getTask("git", "git")).toBeNull();

    submitResult("rex", "review done");
    expect(getTask("git", "git")).toBeNull();

    getTask("sam", "security");
    submitResult("sam", "security done");

    expect(getTask("git", "git")).toEqual(ship);
  });

  it("does not affect tasks with no depends_on", () => {
    loadTasks([pFrontend]);
    expect(getTask("bob", "frontend")).toEqual(pFrontend);
  });
});

describe("job: stageDone", () => {
  it("returns false before any tasks are loaded for the stage", () => {
    expect(stageDone("auth-login", "build")).toBe(false);
  });

  it("returns false when tasks are loaded but none submitted", () => {
    loadTasks([pFrontend]);
    expect(stageDone("auth-login", "build")).toBe(false);
  });

  it("returns true when all tasks in the stage are submitted", () => {
    loadTasks([pFrontend]);
    getTask("bob", "frontend");
    submitResult("bob", "done");
    expect(stageDone("auth-login", "build")).toBe(true);
  });

  it("returns false when only some tasks in the stage are submitted", () => {
    const t2: Task = { id: "p1b", role: "frontend", description: "Build signup form", job: "auth-login", stage: "build" };
    loadTasks([pFrontend, t2]);
    getTask("bob", "frontend");
    submitResult("bob", "done");
    expect(stageDone("auth-login", "build")).toBe(false);
  });
});

describe("job: getStageResults", () => {
  it("returns empty object for unknown stage", () => {
    expect(getStageResults("auth-login", "build")).toEqual({});
  });

  it("returns results keyed by worker id", () => {
    loadTasks([pFrontend]);
    getTask("bob", "frontend");
    submitResult("bob", "login form complete");
    expect(getStageResults("auth-login", "build")).toEqual({ bob: "login form complete" });
  });
});

describe("job: getJobStatus", () => {
  it("returns null for unknown job", () => {
    expect(getJobStatus("unknown")).toBeNull();
  });

  it("shows active once tasks are loaded for a stage", () => {
    loadTasks([pFrontend]);
    expect(getJobStatus("auth-login")!.stages["build"].status).toBe("active");
  });

  it("shows complete stage once all tasks submitted", () => {
    loadTasks([pFrontend]);
    getTask("bob", "frontend");
    submitResult("bob", "done");
    expect(getJobStatus("auth-login")!.stages["build"].status).toBe("complete");
  });
});

describe("job: getAllJobsStatus", () => {
  it("returns all registered jobs", () => {
    const t2: Task = { id: "x1", role: "backend", description: "API work", job: "dashboard", stage: "build" };
    loadTasks([pFrontend, t2]);
    const all = getAllJobsStatus();
    expect(Object.keys(all)).toContain("auth-login");
    expect(Object.keys(all)).toContain("dashboard");
  });

  it("returns empty object when no jobs loaded", () => {
    expect(getAllJobsStatus()).toEqual({});
  });
});

describe("job: resetJob", () => {
  it("returns false for an unknown job", () => {
    expect(resetJob("unknown")).toBe(false);
  });

  it("clears stage state so the job can be rerun", () => {
    loadTasks([pFrontend]);
    getTask("bob", "frontend");
    submitResult("bob", "done");
    expect(stageDone("auth-login", "build")).toBe(true);

    resetJob("auth-login");

    expect(stageDone("auth-login", "build")).toBe(false);
    expect(getJobStatus("auth-login")).toBeNull();
  });

  it("removes queued tasks for the job", () => {
    loadTasks([pFrontend, pReview, pSecurity]);
    getTask("bob", "frontend");
    submitResult("bob", "done");
    // pReview and pSecurity are now in queue (depends_on satisfied)
    resetJob("auth-login");
    expect(getTask("rex", "review")).toBeNull();
    expect(getTask("sam", "security")).toBeNull();
  });

  it("allows new tasks under the same job name after reset", () => {
    loadTasks([pFrontend]);
    getTask("bob", "frontend");
    submitResult("bob", "v1");
    resetJob("auth-login");

    const t2: Task = { id: "p1b", role: "frontend", description: "New feature", job: "auth-login", stage: "build" };
    loadTasks([t2]);
    getTask("alice", "frontend");
    submitResult("alice", "v2");

    expect(stageDone("auth-login", "build")).toBe(true);
    expect(getStageResults("auth-login", "build")).toEqual({ alice: "v2" });
  });
});

describe("reportBlocked / resolveBlock", () => {
  it("sets worker status to blocked with reason", () => {
    loadTasks([pFrontend]);
    getTask("bob", "frontend");
    reportBlocked("bob", "can't find the component");
    const w = getStatus().workers["bob"];
    expect(w.status).toBe("blocked");
    expect(w.blockedReason).toBe("can't find the component");
  });

  it("resolveBlock sets status back to working and clears reason", () => {
    loadTasks([pFrontend]);
    getTask("bob", "frontend");
    reportBlocked("bob", "stuck");
    resolveBlock("bob", "fixed it");
    const w = getStatus().workers["bob"];
    expect(w.status).toBe("working");
    expect(w.blockedReason).toBeUndefined();
  });

  it("allDone returns false when a worker is blocked", () => {
    loadTasks([pFrontend]);
    getTask("bob", "frontend");
    reportBlocked("bob", "stuck");
    expect(allDone()).toBe(false);
  });

  it("resolveBlock writes a knowledge entry to .claude/knowledge/<role>.md", () => {
    const origCwd = process.cwd();
    const tmpDir = mkdtempSync(join(tmpdir(), "state-test-"));
    process.chdir(tmpDir);
    try {
      loadTasks([pFrontend]);
      getTask("bob", "frontend");
      reportBlocked("bob", "missing env var VITE_API_URL");
      resolveBlock("bob", "document required env vars in .env.example before starting");

      const content = readFileSync(".claude/knowledge/frontend.md", "utf8");
      expect(content).toContain("missing env var VITE_API_URL");
      expect(content).toContain("document required env vars");
      expect(content).toContain("job: auth-login");
      expect(content).toContain("stage: build");
    } finally {
      process.chdir(origCwd);
      rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("resolveBlock does nothing when worker is unknown", () => {
    expect(() => resolveBlock("nobody", "resolution")).not.toThrow();
  });
});
