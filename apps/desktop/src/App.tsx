import { useEffect, useMemo, useRef, useState } from "react";
import { invoke } from "@tauri-apps/api/core";
import { listen } from "@tauri-apps/api/event";
import { transitionMode } from "@tabula/domain";
import { Terminal } from "xterm";
import { FitAddon } from "xterm-addon-fit";
import type {
  ArtifactDetail,
  ArtifactSummary,
  CognitiveMode,
  ContextMode,
  ModeState,
  StartSessionResponse,
  SubmitPromptResponse,
  TermOutputEvent,
  TermStatusEvent,
  TurnCompletedEvent,
  WarningEmittedEvent,
  WarningRecord
} from "./types";

export function App() {
  const [contextMode, setContextMode] = useState<ContextMode>("global");
  const [cognitiveMode, setCognitiveMode] = useState<CognitiveMode>("dialogue");
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [sessionProfile, setSessionProfile] = useState<"codex" | "shell">("codex");
  const [sessionRunning, setSessionRunning] = useState(false);
  const [composerInput, setComposerInput] = useState("");
  const [drawerOpen, setDrawerOpen] = useState(true);
  const [statusLine, setStatusLine] = useState("Idle");
  const [artifacts, setArtifacts] = useState<ArtifactSummary[]>([]);
  const [selectedArtifactId, setSelectedArtifactId] = useState<string | null>(null);
  const [artifactDetail, setArtifactDetail] = useState<ArtifactDetail | null>(null);
  const [warnings, setWarnings] = useState<WarningRecord[]>([]);

  const terminalHostRef = useRef<HTMLDivElement | null>(null);
  const terminalRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);

  const modeState: ModeState = useMemo(
    () => ({ context_mode: contextMode, cognitive_mode: cognitiveMode }),
    [contextMode, cognitiveMode]
  );

  useEffect(() => {
    const term = new Terminal({
      convertEol: true,
      cursorBlink: true,
      fontFamily: "Iosevka, IBM Plex Mono, monospace",
      fontSize: 13
    });
    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);

    const mount = terminalHostRef.current;
    if (mount) {
      term.open(mount);
      fitAddon.fit();
    }

    term.onData((data) => {
      if (!sessionId) {
        return;
      }
      void invoke("send_input", {
        session_id: sessionId,
        data
      }).catch((err) => setStatusLine(`terminal input failed: ${String(err)}`));
    });

    terminalRef.current = term;
    fitAddonRef.current = fitAddon;

    const onResize = () => {
      fitAddon.fit();
      if (!sessionId) {
        return;
      }
      const cols = terminalRef.current?.cols ?? 120;
      const rows = terminalRef.current?.rows ?? 40;
      void invoke("resize_session", { session_id: sessionId, cols, rows }).catch(() => undefined);
    };

    window.addEventListener("resize", onResize);

    const unsubsPromise = Promise.all([
      listen<TermOutputEvent>("term_output", (event) => {
        if (!sessionId || event.payload.session_id !== sessionId) {
          return;
        }
        terminalRef.current?.write(event.payload.chunk);
      }),
      listen<TermStatusEvent>("term_status", (event) => {
        if (!sessionId || event.payload.session_id !== sessionId) {
          return;
        }
        setSessionRunning(event.payload.running);
        if (!event.payload.running) {
          setStatusLine(`Session stopped (exit=${event.payload.exit_code ?? "unknown"})`);
        }
      }),
      listen<TurnCompletedEvent>("turn_completed", (event) => {
        setStatusLine(`Turn ${event.payload.turn_id} completed`);
        void refreshArtifacts();
        void refreshWarnings();
      }),
      listen<WarningEmittedEvent>("warning_emitted", (event) => {
        setStatusLine(`${event.payload.code}: ${event.payload.message}`);
      })
    ]);

    return () => {
      window.removeEventListener("resize", onResize);
      void unsubsPromise.then((unsubs) => {
        for (const unlisten of unsubs) {
          unlisten();
        }
      });
      term.dispose();
      terminalRef.current = null;
      fitAddonRef.current = null;
    };
  }, [sessionId]);

  useEffect(() => {
    if (!selectedArtifactId) {
      setArtifactDetail(null);
      return;
    }
    void invoke<ArtifactDetail>("load_artifact", { artifact_id: selectedArtifactId })
      .then((detail) => setArtifactDetail(detail))
      .catch((err) => {
        setStatusLine(`artifact load failed: ${String(err)}`);
        setArtifactDetail(null);
      });
  }, [selectedArtifactId]);

  async function startSession(profile: "codex" | "shell") {
    try {
      const response = await invoke<StartSessionResponse>("start_session", { profile, cwd: null });
      setSessionId(response.session_id);
      setSessionProfile(response.profile);
      setSessionRunning(true);
      setStatusLine(`Session ${response.session_id} started`);
      terminalRef.current?.reset();
      fitAddonRef.current?.fit();
      terminalRef.current?.focus();
      const cols = terminalRef.current?.cols ?? 120;
      const rows = terminalRef.current?.rows ?? 40;
      await invoke("resize_session", { session_id: response.session_id, cols, rows });
    } catch (err) {
      setStatusLine(`start session failed: ${String(err)}`);
    }
  }

  async function stopSession() {
    if (!sessionId) {
      return;
    }
    try {
      await invoke("stop_session", { session_id: sessionId });
      setStatusLine(`Session ${sessionId} stopped`);
      setSessionId(null);
      setSessionRunning(false);
    } catch (err) {
      setStatusLine(`stop session failed: ${String(err)}`);
    }
  }

  async function applyMode(nextContext: ContextMode, nextCognitive: CognitiveMode) {
    const localTransition = transitionMode(
      { contextMode, cognitiveMode },
      { contextMode: nextContext, cognitiveMode: nextCognitive }
    );
    if (!localTransition.ok) {
      setStatusLine(localTransition.message);
      return;
    }

    try {
      await invoke("set_mode", {
        context_mode: nextContext,
        cognitive_mode: nextCognitive
      });
      setContextMode(nextContext);
      setCognitiveMode(nextCognitive);
    } catch (err) {
      setStatusLine(`mode transition rejected: ${String(err)}`);
    }
  }

  async function submitPrompt() {
    const prompt = composerInput.trim();
    if (!prompt || !sessionId) {
      return;
    }
    try {
      const result = await invoke<SubmitPromptResponse>("submit_prompt", {
        input: {
          session_id: sessionId,
          prompt,
          mode: modeState,
          plan_lock_hash: null
        }
      });
      setStatusLine(`Prompt submitted (turn ${result.turn_id})`);
      setComposerInput("");
      await refreshArtifacts();
      await refreshWarnings();
    } catch (err) {
      setStatusLine(`submit failed: ${String(err)}`);
    }
  }

  async function refreshArtifacts() {
    const list = await invoke<ArtifactSummary[]>("list_artifacts", { turn_id: null });
    setArtifacts(list);
    const latest = list[0];
    if (latest) {
      setSelectedArtifactId(latest.id);
    }
  }

  async function refreshWarnings() {
    const list = await invoke<WarningRecord[]>("list_warnings", { turn_id: null });
    setWarnings(list);
  }

  useEffect(() => {
    void refreshArtifacts();
    void refreshWarnings();
  }, []);

  const latestTextArtifact = artifacts.find((a) => a.artifact_type === "text_response") ?? null;
  const canvasState = latestTextArtifact ? "single_artifact" : "blank";

  return (
    <div className="app-shell">
      <header className="top-bar">
        <div className="brand">Tabula</div>
        <div className="mode-chipbar" role="toolbar" aria-label="Mode selector">
          <label>
            Context
            <select
              value={contextMode}
              onChange={(e) => void applyMode(e.target.value as ContextMode, cognitiveMode)}
            >
              <option value="global">global</option>
              <option value="project">project</option>
            </select>
          </label>
          <label>
            Cognitive
            <select
              value={cognitiveMode}
              onChange={(e) => void applyMode(contextMode, e.target.value as CognitiveMode)}
            >
              <option value="dialogue">dialogue</option>
              <option value="plan">plan</option>
              <option value="execution">execution</option>
              <option value="review">review</option>
            </select>
          </label>
        </div>
        <div className="session-state">
          {sessionId ? `${sessionProfile}:${sessionId.slice(0, 8)} ${sessionRunning ? "running" : "stopped"}` : "no session"}
        </div>
      </header>

      <main className="workspace">
        <section className="canvas" aria-live="polite">
          {canvasState === "blank" ? (
            <div className="blank-state">
              <h2>Blank canvas</h2>
              <p>No artifacts yet. Submit a prompt to create canonical text output.</p>
            </div>
          ) : (
            <article className="artifact-card">
              <header>
                <h2>Latest Artifact</h2>
                <div>{latestTextArtifact?.turn_id}</div>
              </header>
              <div className="artifact-tabs">
                {artifacts.map((artifact) => (
                  <button
                    key={artifact.id}
                    className={artifact.id === selectedArtifactId ? "active" : ""}
                    onClick={() => setSelectedArtifactId(artifact.id)}
                  >
                    {artifact.artifact_type}
                  </button>
                ))}
              </div>
              <pre className="artifact-body">{artifactDetail?.body ?? "Select an artifact"}</pre>
            </article>
          )}

          <section className="warnings" aria-live="polite">
            <h3>Warnings</h3>
            {warnings.length === 0 ? (
              <p>No warnings</p>
            ) : (
              <ul>
                {warnings.map((warning) => (
                  <li key={warning.id}>
                    <strong>{warning.code}</strong>: {warning.message}
                  </li>
                ))}
              </ul>
            )}
          </section>
        </section>

        <aside className={`terminal-drawer ${drawerOpen ? "open" : "closed"}`}>
          <div className="drawer-header">
            <h3>Terminal</h3>
            <button onClick={() => setDrawerOpen((v) => !v)}>{drawerOpen ? "Hide" : "Show"}</button>
          </div>
          {drawerOpen && (
            <>
              <div className="drawer-actions">
                <button onClick={() => void startSession("codex")} disabled={Boolean(sessionId && sessionRunning)}>
                  Start Codex
                </button>
                <button onClick={() => void startSession("shell")} disabled={Boolean(sessionId && sessionRunning)}>
                  Start Shell
                </button>
                <button onClick={() => void stopSession()} disabled={!sessionId}>
                  Stop Session
                </button>
              </div>
              <div className="terminal-host" ref={terminalHostRef} />
            </>
          )}
        </aside>
      </main>

      <footer className="composer">
        <input
          value={composerInput}
          onChange={(e) => setComposerInput(e.target.value)}
          placeholder="Type prompt and press Enter"
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              void submitPrompt();
            }
          }}
        />
        <button onClick={() => void submitPrompt()} disabled={!sessionId || !composerInput.trim()}>
          Submit
        </button>
        <div className="status-line" role="status">
          {statusLine}
        </div>
      </footer>
    </div>
  );
}
