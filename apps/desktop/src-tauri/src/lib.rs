use std::collections::HashMap;
use std::fs;
use std::io::{Read, Write};
use std::path::{Path, PathBuf};
use std::sync::Mutex;

use chrono::Utc;
use portable_pty::{native_pty_system, Child, CommandBuilder, MasterPty, PtySize};
use rusqlite::{params, Connection};
use serde::{Deserialize, Serialize};
use tauri::{AppHandle, Emitter, State};
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
struct ModeState {
    context_mode: String,
    cognitive_mode: String,
}

#[derive(Debug, Clone, Serialize)]
struct StartSessionResponse {
    session_id: String,
    profile: String,
}

#[derive(Debug, Clone, Serialize)]
struct SubmitPromptResponse {
    turn_id: String,
    artifact_ids: Vec<String>,
    warning_ids: Vec<String>,
}

#[derive(Debug, Clone, Serialize)]
struct ArtifactSummary {
    id: String,
    turn_id: String,
    artifact_type: String,
    storage_path: String,
    created_at: String,
}

#[derive(Debug, Clone, Serialize)]
struct ArtifactDetail {
    id: String,
    turn_id: String,
    artifact_type: String,
    storage_path: String,
    created_at: String,
    body: String,
}

#[derive(Debug, Clone, Serialize)]
struct WarningRecord {
    id: String,
    turn_id: String,
    code: String,
    message: String,
    severity: String,
    created_at: String,
}

#[derive(Debug, Clone, Serialize)]
struct TurnCompletedEvent {
    turn_id: String,
    artifact_ids: Vec<String>,
    warning_ids: Vec<String>,
}

#[derive(Debug, Clone, Serialize)]
struct WarningEmittedEvent {
    turn_id: String,
    code: String,
    message: String,
    severity: String,
}

#[derive(Debug, Clone, Serialize)]
struct TermOutputEvent {
    session_id: String,
    chunk: String,
}

#[derive(Debug, Clone, Serialize)]
struct TermStatusEvent {
    session_id: String,
    running: bool,
    exit_code: Option<i32>,
}

#[derive(Debug, Clone, Deserialize)]
struct SubmitPromptInput {
    session_id: String,
    prompt: String,
    mode: ModeState,
    plan_lock_hash: Option<String>,
}

struct SessionHandle {
    master: Box<dyn MasterPty + Send>,
    writer: Box<dyn Write + Send>,
    child: Box<dyn Child + Send>,
}

struct AppState {
    sessions: Mutex<HashMap<String, SessionHandle>>,
    mode: Mutex<ModeState>,
    db_path: PathBuf,
    artifacts_root: PathBuf,
}

fn write_session_input(state: &State<'_, AppState>, session_id: &str, data: &str) -> Result<(), String> {
    let mut sessions = state
        .sessions
        .lock()
        .map_err(|_| "session lock poisoned".to_string())?;
    let session = sessions
        .get_mut(session_id)
        .ok_or_else(|| format!("session not found: {session_id}"))?;

    session
        .writer
        .write_all(data.as_bytes())
        .map_err(|e| format!("write to pty failed: {e}"))?;
    session
        .writer
        .flush()
        .map_err(|e| format!("flush pty failed: {e}"))?;

    insert_session_event(
        &state.db_path,
        session_id,
        "input",
        &serde_json::json!({ "data": data }).to_string(),
    )?;
    Ok(())
}

fn now_rfc3339() -> String {
    Utc::now().to_rfc3339()
}

fn tabula_root() -> Result<PathBuf, String> {
    let home = dirs::home_dir().ok_or_else(|| "could not resolve home directory".to_string())?;
    Ok(home.join(".tabula"))
}

fn init_storage() -> Result<(PathBuf, PathBuf), String> {
    let root = tabula_root()?;
    let artifacts = root.join("artifacts");
    fs::create_dir_all(&artifacts).map_err(|e| format!("failed to create artifacts dir: {e}"))?;
    Ok((root.join("state.db"), artifacts))
}

fn open_conn(path: &Path) -> Result<Connection, String> {
    Connection::open(path).map_err(|e| format!("sqlite open failed: {e}"))
}

fn init_schema(path: &Path) -> Result<(), String> {
    let conn = open_conn(path)?;
    conn.execute_batch(
        r#"
        CREATE TABLE IF NOT EXISTS sessions (
            id TEXT PRIMARY KEY,
            profile TEXT NOT NULL,
            cwd TEXT,
            created_at TEXT NOT NULL,
            closed_at TEXT,
            last_status TEXT
        );

        CREATE TABLE IF NOT EXISTS session_events (
            id TEXT PRIMARY KEY,
            session_id TEXT NOT NULL,
            seq INTEGER NOT NULL,
            kind TEXT NOT NULL,
            payload_json TEXT NOT NULL,
            created_at TEXT NOT NULL
        );

        CREATE TABLE IF NOT EXISTS turns (
            id TEXT PRIMARY KEY,
            session_id TEXT NOT NULL,
            user_prompt TEXT NOT NULL,
            mode_context TEXT NOT NULL,
            mode_cognitive TEXT NOT NULL,
            created_at TEXT NOT NULL
        );

        CREATE TABLE IF NOT EXISTS artifacts (
            id TEXT PRIMARY KEY,
            turn_id TEXT NOT NULL,
            artifact_type TEXT NOT NULL,
            storage_path TEXT NOT NULL,
            created_at TEXT NOT NULL
        );

        CREATE TABLE IF NOT EXISTS warnings (
            id TEXT PRIMARY KEY,
            turn_id TEXT NOT NULL,
            code TEXT NOT NULL,
            message TEXT NOT NULL,
            severity TEXT NOT NULL,
            created_at TEXT NOT NULL
        );

        CREATE TABLE IF NOT EXISTS app_state (
            key TEXT PRIMARY KEY,
            value_json TEXT NOT NULL,
            updated_at TEXT NOT NULL
        );
        "#,
    )
    .map_err(|e| format!("sqlite schema init failed: {e}"))?;

    Ok(())
}

fn is_patch_like(content: &str) -> bool {
    let trimmed = content.trim_start();
    trimmed.starts_with("diff --git")
        || trimmed.starts_with("--- ")
        || content.contains("\n+++ ")
        || content.contains("\n@@")
}

fn mode_transition_allowed(previous: &ModeState, next: &ModeState) -> bool {
    match previous.cognitive_mode.as_str() {
        "dialogue" => matches!(next.cognitive_mode.as_str(), "dialogue" | "plan"),
        "plan" => matches!(next.cognitive_mode.as_str(), "plan" | "dialogue" | "execution"),
        "execution" => matches!(next.cognitive_mode.as_str(), "execution" | "review"),
        "review" => matches!(next.cognitive_mode.as_str(), "review" | "dialogue" | "plan"),
        _ => false,
    }
}

fn evaluate_warnings(mode: &ModeState, prompt: &str, plan_lock_hash: Option<&str>) -> Vec<(String, String, String)> {
    let mut out: Vec<(String, String, String)> = Vec::new();

    if mode.context_mode == "project" && mode.cognitive_mode == "execution" {
        let has_lock = plan_lock_hash.map(|v| !v.trim().is_empty()).unwrap_or(false);
        if !has_lock {
            out.push((
                "plan_lock_missing".to_string(),
                "Execution mode is active without a plan lock hash. This is advisory only in v1.".to_string(),
                "warning".to_string(),
            ));
        }
    }

    let lower = prompt.to_ascii_lowercase();
    let risky = ["rm -rf", "sudo", "chmod", "chown", "truncate", " mv ", " cp ", " tee "];
    if mode.context_mode == "global" && risky.iter().any(|needle| lower.contains(needle)) {
        out.push((
            "global_write_risk".to_string(),
            "Prompt appears to include risky global write/exec intent. Advisory warning emitted.".to_string(),
            "warning".to_string(),
        ));
    }

    out
}

fn command_for_profile(profile: &str) -> Result<CommandBuilder, String> {
    match profile {
        "codex" => {
            let mut cmd = CommandBuilder::new("bash");
            cmd.arg("-ilc");
            cmd.arg("if command -v codex >/dev/null 2>&1; then exec codex; else echo 'codex not found; starting shell'; exec bash -li; fi");
            Ok(cmd)
        }
        "shell" => {
            let mut cmd = CommandBuilder::new("bash");
            cmd.arg("-li");
            Ok(cmd)
        }
        other => Err(format!("unsupported profile '{other}' (allowed: codex, shell)")),
    }
}

fn insert_session_record(db_path: &Path, session_id: &str, profile: &str, cwd: Option<&str>) -> Result<(), String> {
    let conn = open_conn(db_path)?;
    conn.execute(
        "INSERT INTO sessions (id, profile, cwd, created_at, last_status) VALUES (?1, ?2, ?3, ?4, ?5)",
        params![session_id, profile, cwd, now_rfc3339(), "running"],
    )
    .map_err(|e| format!("insert session failed: {e}"))?;
    Ok(())
}

fn close_session_record(db_path: &Path, session_id: &str, last_status: &str) -> Result<(), String> {
    let conn = open_conn(db_path)?;
    conn.execute(
        "UPDATE sessions SET closed_at = ?2, last_status = ?3 WHERE id = ?1",
        params![session_id, now_rfc3339(), last_status],
    )
    .map_err(|e| format!("close session update failed: {e}"))?;
    Ok(())
}

fn insert_session_event(db_path: &Path, session_id: &str, kind: &str, payload_json: &str) -> Result<(), String> {
    let conn = open_conn(db_path)?;
    conn.execute(
        "INSERT INTO session_events (id, session_id, seq, kind, payload_json, created_at) VALUES (?1, ?2, ?3, ?4, ?5, ?6)",
        params![Uuid::new_v4().to_string(), session_id, 0_i64, kind, payload_json, now_rfc3339()],
    )
    .map_err(|e| format!("insert session event failed: {e}"))?;
    Ok(())
}

fn spawn_output_pump(app: AppHandle, session_id: String, mut reader: Box<dyn Read + Send>) {
    std::thread::spawn(move || {
        let mut buf = [0_u8; 4096];
        loop {
            match reader.read(&mut buf) {
                Ok(0) => {
                    let _ = app.emit(
                        "term_status",
                        TermStatusEvent {
                            session_id: session_id.clone(),
                            running: false,
                            exit_code: None,
                        },
                    );
                    break;
                }
                Ok(n) => {
                    let chunk = String::from_utf8_lossy(&buf[..n]).to_string();
                    let _ = app.emit(
                        "term_output",
                        TermOutputEvent {
                            session_id: session_id.clone(),
                            chunk,
                        },
                    );
                }
                Err(_) => {
                    let _ = app.emit(
                        "term_status",
                        TermStatusEvent {
                            session_id: session_id.clone(),
                            running: false,
                            exit_code: None,
                        },
                    );
                    break;
                }
            }
        }
    });
}

#[tauri::command]
fn start_session(
    app: AppHandle,
    state: State<'_, AppState>,
    profile: String,
    cwd: Option<String>,
) -> Result<StartSessionResponse, String> {
    let pty_system = native_pty_system();
    let pair = pty_system
        .openpty(PtySize {
            rows: 40,
            cols: 120,
            pixel_width: 0,
            pixel_height: 0,
        })
        .map_err(|e| format!("open pty failed: {e}"))?;

    let mut cmd = command_for_profile(&profile)?;
    if let Some(dir) = cwd.as_ref().filter(|v| !v.trim().is_empty()) {
        cmd.cwd(dir);
    }

    let child = pair
        .slave
        .spawn_command(cmd)
        .map_err(|e| format!("spawn command failed: {e}"))?;

    let writer = pair
        .master
        .take_writer()
        .map_err(|e| format!("take writer failed: {e}"))?;

    let reader = pair
        .master
        .try_clone_reader()
        .map_err(|e| format!("clone reader failed: {e}"))?;

    let session_id = Uuid::new_v4().to_string();
    spawn_output_pump(app.clone(), session_id.clone(), reader);

    {
        let mut sessions = state.sessions.lock().map_err(|_| "session lock poisoned".to_string())?;
        sessions.insert(
            session_id.clone(),
            SessionHandle {
                master: pair.master,
                writer,
                child,
            },
        );
    }

    insert_session_record(&state.db_path, &session_id, &profile, cwd.as_deref())?;

    app.emit(
        "term_status",
        TermStatusEvent {
            session_id: session_id.clone(),
            running: true,
            exit_code: None,
        },
    )
    .map_err(|e| format!("emit term_status failed: {e}"))?;

    Ok(StartSessionResponse { session_id, profile })
}

#[tauri::command]
fn send_input(state: State<'_, AppState>, session_id: String, data: String) -> Result<(), String> {
    write_session_input(&state, &session_id, &data)
}

#[tauri::command]
fn resize_session(state: State<'_, AppState>, session_id: String, cols: u16, rows: u16) -> Result<(), String> {
    let mut sessions = state.sessions.lock().map_err(|_| "session lock poisoned".to_string())?;
    let session = sessions
        .get_mut(&session_id)
        .ok_or_else(|| format!("session not found: {session_id}"))?;

    session
        .master
        .resize(PtySize {
            rows,
            cols,
            pixel_width: 0,
            pixel_height: 0,
        })
        .map_err(|e| format!("resize failed: {e}"))
}

#[tauri::command]
fn stop_session(app: AppHandle, state: State<'_, AppState>, session_id: String) -> Result<(), String> {
    let mut sessions = state.sessions.lock().map_err(|_| "session lock poisoned".to_string())?;
    let mut session = sessions
        .remove(&session_id)
        .ok_or_else(|| format!("session not found: {session_id}"))?;

    let _ = session.child.kill();
    let _ = session.child.wait();

    close_session_record(&state.db_path, &session_id, "stopped")?;

    app.emit(
        "term_status",
        TermStatusEvent {
            session_id,
            running: false,
            exit_code: None,
        },
    )
    .map_err(|e| format!("emit term_status failed: {e}"))?;

    Ok(())
}

#[tauri::command]
fn set_mode(
    app: AppHandle,
    state: State<'_, AppState>,
    context_mode: String,
    cognitive_mode: String,
) -> Result<ModeState, String> {
    let next = ModeState {
        context_mode,
        cognitive_mode,
    };

    let mut mode = state.mode.lock().map_err(|_| "mode lock poisoned".to_string())?;
    if !mode_transition_allowed(&mode, &next) {
        return Err(format!(
            "invalid cognitive transition {} -> {}",
            mode.cognitive_mode, next.cognitive_mode
        ));
    }

    *mode = next.clone();

    let conn = open_conn(&state.db_path)?;
    conn.execute(
        "INSERT INTO app_state (key, value_json, updated_at) VALUES ('mode', ?1, ?2)
         ON CONFLICT(key) DO UPDATE SET value_json = excluded.value_json, updated_at = excluded.updated_at",
        params![serde_json::to_string(&next).map_err(|e| format!("serialize mode failed: {e}"))?, now_rfc3339()],
    )
    .map_err(|e| format!("persist mode failed: {e}"))?;

    app.emit("mode_changed", &next)
        .map_err(|e| format!("emit mode_changed failed: {e}"))?;

    Ok(next)
}

#[tauri::command]
fn submit_prompt(app: AppHandle, state: State<'_, AppState>, input: SubmitPromptInput) -> Result<SubmitPromptResponse, String> {
    let prompt = input.prompt.trim();
    if prompt.is_empty() {
        return Err("prompt must not be empty".to_string());
    }

    write_session_input(&state, &input.session_id, &format!("{prompt}\n"))?;

    let turn_id = Uuid::new_v4().to_string();
    let created_at = now_rfc3339();

    let conn = open_conn(&state.db_path)?;
    conn.execute(
        "INSERT INTO turns (id, session_id, user_prompt, mode_context, mode_cognitive, created_at) VALUES (?1, ?2, ?3, ?4, ?5, ?6)",
        params![
            turn_id,
            input.session_id,
            prompt,
            input.mode.context_mode,
            input.mode.cognitive_mode,
            created_at
        ],
    )
    .map_err(|e| format!("insert turn failed: {e}"))?;

    let turn_dir = state.artifacts_root.join(&turn_id);
    fs::create_dir_all(&turn_dir).map_err(|e| format!("create turn artifact dir failed: {e}"))?;

    let mut artifact_ids = Vec::new();

    let text_artifact_id = Uuid::new_v4().to_string();
    let text_path = turn_dir.join("canonical-text.md");
    let text_body = format!(
        "# Turn {}\n\n- context: {}\n- cognitive: {}\n- session: {}\n\n## Prompt\n\n{}\n",
        turn_id, input.mode.context_mode, input.mode.cognitive_mode, input.session_id, prompt
    );
    fs::write(&text_path, text_body).map_err(|e| format!("write canonical-text failed: {e}"))?;

    conn.execute(
        "INSERT INTO artifacts (id, turn_id, artifact_type, storage_path, created_at) VALUES (?1, ?2, 'text_response', ?3, ?4)",
        params![text_artifact_id, turn_id, text_path.to_string_lossy(), now_rfc3339()],
    )
    .map_err(|e| format!("insert text artifact failed: {e}"))?;
    artifact_ids.push(text_artifact_id);

    if is_patch_like(prompt) {
        let diff_artifact_id = Uuid::new_v4().to_string();
        let diff_path = turn_dir.join("latest.patch");
        fs::write(&diff_path, prompt).map_err(|e| format!("write diff artifact failed: {e}"))?;
        conn.execute(
            "INSERT INTO artifacts (id, turn_id, artifact_type, storage_path, created_at) VALUES (?1, ?2, 'diff_patch', ?3, ?4)",
            params![diff_artifact_id, turn_id, diff_path.to_string_lossy(), now_rfc3339()],
        )
        .map_err(|e| format!("insert diff artifact failed: {e}"))?;
        artifact_ids.push(diff_artifact_id);
    }

    let warnings = evaluate_warnings(&input.mode, prompt, input.plan_lock_hash.as_deref());
    let mut warning_ids = Vec::new();

    for (code, message, severity) in warnings {
        let warning_id = Uuid::new_v4().to_string();
        conn.execute(
            "INSERT INTO warnings (id, turn_id, code, message, severity, created_at) VALUES (?1, ?2, ?3, ?4, ?5, ?6)",
            params![warning_id, turn_id, code, message, severity, now_rfc3339()],
        )
        .map_err(|e| format!("insert warning failed: {e}"))?;

        app.emit(
            "warning_emitted",
            WarningEmittedEvent {
                turn_id: turn_id.clone(),
                code: code.clone(),
                message: message.clone(),
                severity,
            },
        )
        .map_err(|e| format!("emit warning failed: {e}"))?;

        warning_ids.push(warning_id);
    }

    app.emit(
        "turn_completed",
        TurnCompletedEvent {
            turn_id: turn_id.clone(),
            artifact_ids: artifact_ids.clone(),
            warning_ids: warning_ids.clone(),
        },
    )
    .map_err(|e| format!("emit turn_completed failed: {e}"))?;

    Ok(SubmitPromptResponse {
        turn_id,
        artifact_ids,
        warning_ids,
    })
}

#[tauri::command]
fn list_artifacts(state: State<'_, AppState>, turn_id: Option<String>) -> Result<Vec<ArtifactSummary>, String> {
    let conn = open_conn(&state.db_path)?;

    let mut out = Vec::new();
    if let Some(turn_id) = turn_id {
        let mut stmt = conn
            .prepare(
                "SELECT id, turn_id, artifact_type, storage_path, created_at
                 FROM artifacts WHERE turn_id = ?1 ORDER BY created_at DESC",
            )
            .map_err(|e| format!("prepare artifacts query failed: {e}"))?;

        let rows = stmt
            .query_map(params![turn_id], |row| {
                Ok(ArtifactSummary {
                    id: row.get(0)?,
                    turn_id: row.get(1)?,
                    artifact_type: row.get(2)?,
                    storage_path: row.get(3)?,
                    created_at: row.get(4)?,
                })
            })
            .map_err(|e| format!("query artifacts failed: {e}"))?;

        for row in rows {
            out.push(row.map_err(|e| format!("read artifact row failed: {e}"))?);
        }
    } else {
        let mut stmt = conn
            .prepare(
                "SELECT id, turn_id, artifact_type, storage_path, created_at
                 FROM artifacts ORDER BY created_at DESC LIMIT 100",
            )
            .map_err(|e| format!("prepare artifacts query failed: {e}"))?;

        let rows = stmt
            .query_map([], |row| {
                Ok(ArtifactSummary {
                    id: row.get(0)?,
                    turn_id: row.get(1)?,
                    artifact_type: row.get(2)?,
                    storage_path: row.get(3)?,
                    created_at: row.get(4)?,
                })
            })
            .map_err(|e| format!("query artifacts failed: {e}"))?;

        for row in rows {
            out.push(row.map_err(|e| format!("read artifact row failed: {e}"))?);
        }
    }

    Ok(out)
}

#[tauri::command]
fn load_artifact(state: State<'_, AppState>, artifact_id: String) -> Result<ArtifactDetail, String> {
    let conn = open_conn(&state.db_path)?;
    let mut stmt = conn
        .prepare(
            "SELECT id, turn_id, artifact_type, storage_path, created_at FROM artifacts WHERE id = ?1 LIMIT 1",
        )
        .map_err(|e| format!("prepare load artifact failed: {e}"))?;

    let summary = stmt
        .query_row(params![artifact_id], |row| {
            Ok(ArtifactSummary {
                id: row.get(0)?,
                turn_id: row.get(1)?,
                artifact_type: row.get(2)?,
                storage_path: row.get(3)?,
                created_at: row.get(4)?,
            })
        })
        .map_err(|e| format!("artifact not found: {e}"))?;

    let body = fs::read_to_string(&summary.storage_path)
        .map_err(|e| format!("read artifact body failed: {e}"))?;

    Ok(ArtifactDetail {
        id: summary.id,
        turn_id: summary.turn_id,
        artifact_type: summary.artifact_type,
        storage_path: summary.storage_path,
        created_at: summary.created_at,
        body,
    })
}

#[tauri::command]
fn list_warnings(state: State<'_, AppState>, turn_id: Option<String>) -> Result<Vec<WarningRecord>, String> {
    let conn = open_conn(&state.db_path)?;
    let mut out = Vec::new();

    if let Some(turn_id) = turn_id {
        let mut stmt = conn
            .prepare(
                "SELECT id, turn_id, code, message, severity, created_at
                 FROM warnings WHERE turn_id = ?1 ORDER BY created_at DESC",
            )
            .map_err(|e| format!("prepare warnings query failed: {e}"))?;

        let rows = stmt
            .query_map(params![turn_id], |row| {
                Ok(WarningRecord {
                    id: row.get(0)?,
                    turn_id: row.get(1)?,
                    code: row.get(2)?,
                    message: row.get(3)?,
                    severity: row.get(4)?,
                    created_at: row.get(5)?,
                })
            })
            .map_err(|e| format!("query warnings failed: {e}"))?;

        for row in rows {
            out.push(row.map_err(|e| format!("read warning row failed: {e}"))?);
        }
    } else {
        let mut stmt = conn
            .prepare(
                "SELECT id, turn_id, code, message, severity, created_at
                 FROM warnings ORDER BY created_at DESC LIMIT 200",
            )
            .map_err(|e| format!("prepare warnings query failed: {e}"))?;

        let rows = stmt
            .query_map([], |row| {
                Ok(WarningRecord {
                    id: row.get(0)?,
                    turn_id: row.get(1)?,
                    code: row.get(2)?,
                    message: row.get(3)?,
                    severity: row.get(4)?,
                    created_at: row.get(5)?,
                })
            })
            .map_err(|e| format!("query warnings failed: {e}"))?;

        for row in rows {
            out.push(row.map_err(|e| format!("read warning row failed: {e}"))?);
        }
    }

    Ok(out)
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let (db_path, artifacts_root) = match init_storage() {
        Ok(v) => v,
        Err(e) => {
            eprintln!("storage init failed: {e}");
            return;
        }
    };

    if let Err(e) = init_schema(&db_path) {
        eprintln!("schema init failed: {e}");
        return;
    }

    let mode = ModeState {
        context_mode: "global".to_string(),
        cognitive_mode: "dialogue".to_string(),
    };

    tauri::Builder::default()
        .manage(AppState {
            sessions: Mutex::new(HashMap::new()),
            mode: Mutex::new(mode),
            db_path,
            artifacts_root,
        })
        .invoke_handler(tauri::generate_handler![
            start_session,
            send_input,
            resize_session,
            stop_session,
            set_mode,
            submit_prompt,
            list_artifacts,
            load_artifact,
            list_warnings,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

#[cfg(test)]
mod tests {
    use super::{evaluate_warnings, is_patch_like, mode_transition_allowed, ModeState};

    #[test]
    fn patch_detection_works() {
        assert!(is_patch_like("diff --git a/file b/file\n@@"));
        assert!(!is_patch_like("plain text"));
    }

    #[test]
    fn mode_transition_rules() {
        let prev = ModeState {
            context_mode: "global".to_string(),
            cognitive_mode: "dialogue".to_string(),
        };
        let next_ok = ModeState {
            context_mode: "project".to_string(),
            cognitive_mode: "plan".to_string(),
        };
        let next_bad = ModeState {
            context_mode: "project".to_string(),
            cognitive_mode: "execution".to_string(),
        };
        assert!(mode_transition_allowed(&prev, &next_ok));
        assert!(!mode_transition_allowed(&prev, &next_bad));
    }

    #[test]
    fn warnings_are_non_blocking_advisories() {
        let mode = ModeState {
            context_mode: "project".to_string(),
            cognitive_mode: "execution".to_string(),
        };
        let warnings = evaluate_warnings(&mode, "run tests", None);
        assert!(warnings.iter().any(|w| w.0 == "plan_lock_missing"));
    }
}
