import * as env from './app-env.js';
import * as context from './app-context.js';

const { marked, wsURL, renderCanvas, clearCanvas, getLocationFromSelection, clearLineHighlight, escapeHtml, sanitizeHtml, getActiveArtifactTitle, getActiveTextEventId, getPreviousArtifactText, getUiState, setUiMode, showIndicatorMode, hideIndicator, showTextInput, hideTextInput, showOverlay, hideOverlay, updateOverlay, isOverlayVisible, isTextInputVisible, isRecording, setRecording, getInputAnchor, setInputAnchor, pinCursorAnchor, getAnchorFromPoint, buildContextPrefix, getLastInputPosition, setLastInputPosition, configureLiveSession, getLiveSessionSnapshot, handleLiveSessionMessage, isLiveSessionListenActive, LIVE_SESSION_HOTWORD_DEFAULT, LIVE_SESSION_MODE_DIALOGUE, LIVE_SESSION_MODE_MEETING, onLiveSessionTTSPlaybackComplete, cancelLiveSessionListen, startLiveSession, stopLiveSession, initHotword, startHotwordMonitor, stopHotwordMonitor, isHotwordActive, onHotwordDetected, setHotwordThreshold, setHotwordAudioContext, getPreRollAudio, getHotwordMicStream, initVAD, float32ToWav } = env;
const { refs, state, getState, isVoiceTurn, COMPANION_VIEW_PATH_PREFIX, COMPANION_TRANSCRIPT_VIEW_PATH, COMPANION_SUMMARY_VIEW_PATH, COMPANION_REFERENCES_VIEW_PATH, MEETING_TRANSCRIPT_LABEL, MEETING_SUMMARY_LABEL, MEETING_REFERENCES_LABEL, MEETING_SUMMARY_ITEMS_PANEL_ID, CHAT_CTRL_LONG_PRESS_MS, ARTIFACT_EDIT_LONG_TAP_MS, ITEM_SIDEBAR_VIEWS, ITEM_SIDEBAR_GESTURE_CANCEL_PX, ITEM_SIDEBAR_GESTURE_COMMIT_PX, ITEM_SIDEBAR_GESTURE_LONG_PX, ITEM_SIDEBAR_DEFAULT_LATER_HOUR_UTC, ITEM_SIDEBAR_MENU_ID, DEV_UI_RELOAD_POLL_MS, ASSISTANT_ACTIVITY_POLL_MS, CHAT_WS_STALE_THRESHOLD_MS, ACTIVE_TURN_NO_ID_CLEAR_GRACE_MS, ACTIVE_TURN_ACTIVITY_CLEAR_GRACE_MS, PROJECT_CHAT_MODEL_ALIASES, PROJECT_CHAT_MODEL_REASONING_EFFORTS, TTS_SILENT_STORAGE_KEY, YOLO_MODE_STORAGE_KEY, SOMEDAY_REVIEW_NUDGE_ENABLED_STORAGE_KEY, SOMEDAY_REVIEW_NUDGE_LAST_SHOWN_STORAGE_KEY, SOMEDAY_REVIEW_NUDGE_INTERVAL_MS, ACTIVE_PROJECT_STORAGE_KEY, LAST_VIEW_STORAGE_KEY, RUNTIME_RELOAD_CONTEXT_STORAGE_KEY, SIDEBAR_IMAGE_EXTENSIONS, PANEL_MOTION_WATCH_QUERIES, VOICE_LIFECYCLE, COMPANION_IDLE_SURFACES, COMPANION_RUNTIME_STATES, TOOL_PALETTE_MODES } = context;

const showStatus = (...args) => refs.showStatus(...args);
const updateAssistantActivityIndicator = (...args) => refs.updateAssistantActivityIndicator(...args);
const setYoloModeLocal = (...args) => refs.setYoloModeLocal(...args);
const readYoloModePreference = (...args) => refs.readYoloModePreference(...args);
const readSomedayReviewNudgePreference = (...args) => refs.readSomedayReviewNudgePreference(...args);
const setSomedayReviewNudgeEnabled = (...args) => refs.setSomedayReviewNudgeEnabled(...args);
const clearInkDraft = (...args) => refs.clearInkDraft(...args);
const renderInkControls = (...args) => refs.renderInkControls(...args);
const switchProject = (...args) => refs.switchProject(...args);
const setPrReviewDrawerOpen = (...args) => refs.setPrReviewDrawerOpen(...args);
const activeProject = (...args) => refs.activeProject(...args);
const handleStopAction = (...args) => refs.handleStopAction(...args);
const hideCanvasColumn = (...args) => refs.hideCanvasColumn(...args);
const submitMessage = (...args) => refs.submitMessage(...args);
const stopVoiceCaptureAndSend = (...args) => refs.stopVoiceCaptureAndSend(...args);
const cancelChatVoiceCapture = (...args) => refs.cancelChatVoiceCapture(...args);
const handleItemSidebarKeyboardShortcut = (...args) => refs.handleItemSidebarKeyboardShortcut(...args);
const setTTSSilentMode = (...args) => refs.setTTSSilentMode(...args);
const isMobileSilent = (...args) => refs.isMobileSilent(...args);
const restoreRuntimeReloadContext = (...args) => refs.restoreRuntimeReloadContext(...args);
const consumeRuntimeReloadContext = (...args) => refs.consumeRuntimeReloadContext(...args);
const fetchRuntimeMeta = (...args) => refs.fetchRuntimeMeta(...args);
const applyRuntimePreferences = (...args) => refs.applyRuntimePreferences(...args);
const initHotwordLifecycle = (...args) => refs.initHotwordLifecycle(...args);
const resolveInitialProjectID = (...args) => refs.resolveInitialProjectID(...args);
const applyRuntimeReasoningEffortOptions = (...args) => refs.applyRuntimeReasoningEffortOptions(...args);
const fetchProjects = (...args) => refs.fetchProjects(...args);
const startRuntimeReloadWatcher = (...args) => refs.startRuntimeReloadWatcher(...args);
const startAssistantActivityWatcher = (...args) => refs.startAssistantActivityWatcher(...args);
const closeEdgePanels = (...args) => refs.closeEdgePanels(...args);
const syncInteractionBodyState = (...args) => refs.syncInteractionBodyState(...args);
const settleKeyboardAfterSubmit = (...args) => refs.settleKeyboardAfterSubmit(...args);
const ensureArtifactEditor = (...args) => refs.ensureArtifactEditor(...args);
const exitArtifactEditMode = (...args) => refs.exitArtifactEditMode(...args);
const enterArtifactEditMode = (...args) => refs.enterArtifactEditMode(...args);
const canEnterArtifactEditModeFromTarget = (...args) => refs.canEnterArtifactEditModeFromTarget(...args);
const showDisclaimerModal = (...args) => refs.showDisclaimerModal(...args);
const applyIPhoneFrameCorners = (...args) => refs.applyIPhoneFrameCorners(...args);
const initPanelMotionMode = (...args) => refs.initPanelMotionMode(...args);
const initEdgePanels = (...args) => refs.initEdgePanels(...args);
const setPenInkingState = (...args) => refs.setPenInkingState(...args);
const syncInkLayerSize = (...args) => refs.syncInkLayerSize(...args);
const beginVoiceCapture = (...args) => refs.beginVoiceCapture(...args);
const openComposerAt = (...args) => refs.openComposerAt(...args);
const suppressSyntheticClick = (...args) => refs.suppressSyntheticClick(...args);
const isSuppressedClick = (...args) => refs.isSuppressedClick(...args);
const isInkTool = (...args) => refs.isInkTool(...args);
const isEditableTarget = (...args) => refs.isEditableTarget(...args);
const isUiStopGestureActive = (...args) => refs.isUiStopGestureActive(...args);
const isLikelyIOS = (...args) => refs.isLikelyIOS(...args);
const isMobileViewport = (...args) => refs.isMobileViewport(...args);
const stepCanvasFile = (...args) => refs.stepCanvasFile(...args);
const beginInkStroke = (...args) => refs.beginInkStroke(...args);
const extendInkStroke = (...args) => refs.extendInkStroke(...args);
const finalizeInkStroke = (...args) => refs.finalizeInkStroke(...args);
const resetInkDraftState = (...args) => refs.resetInkDraftState(...args);
const getEdgeTapSizePx = (...args) => refs.getEdgeTapSizePx(...args);
const getTopEdgeTapSizePx = (...args) => refs.getTopEdgeTapSizePx(...args);
const prefersTextComposer = (...args) => refs.prefersTextComposer(...args);
const createPdfStickyNoteAt = (...args) => refs.createPdfStickyNoteAt(...args);
const selectInteractionTool = (...args) => refs.selectInteractionTool(...args);
const submitInkDraft = (...args) => refs.submitInkDraft(...args);
const shouldStopInUiClick = (...args) => refs.shouldStopInUiClick(...args);
const hideItemSidebarMenu = (...args) => refs.hideItemSidebarMenu(...args);
const stepPrReviewFile = (...args) => refs.stepPrReviewFile(...args);
const maybeApplySelectionHighlight = (...args) => refs.maybeApplySelectionHighlight(...args);
const openItemSidebarView = (...args) => refs.openItemSidebarView(...args);
const launchNewMailAuthoring = (...args) => refs.launchNewMailAuthoring(...args);
const launchReplyAuthoring = (...args) => refs.launchReplyAuthoring(...args);

const COMMAND_CENTER_ID = 'command-center';
const COMMAND_CENTER_INPUT_ID = 'command-center-input';
const COMMAND_CENTER_LIST_ID = 'command-center-list';
const COMMAND_CENTER_COMMANDS = [
  {
    id: 'view-inbox',
    title: 'Open Inbox',
    detail: 'Show inbox items in the left sidebar.',
    shortcut: 'Inbox',
    keywords: 'inbox mail tasks items',
    run: () => openItemSidebarView('inbox'),
  },
  {
    id: 'view-waiting',
    title: 'Open Waiting',
    detail: 'Show waiting items in the left sidebar.',
    shortcut: 'Waiting',
    keywords: 'waiting follow up items',
    run: () => openItemSidebarView('waiting'),
  },
  {
    id: 'view-someday',
    title: 'Open Someday',
    detail: 'Show someday items in the left sidebar.',
    shortcut: 'Someday',
    keywords: 'someday backlog items',
    run: () => openItemSidebarView('someday'),
  },
  {
    id: 'view-done',
    title: 'Open Done',
    detail: 'Show completed items in the left sidebar.',
    shortcut: 'Done',
    keywords: 'done completed archive items',
    run: () => openItemSidebarView('done'),
  },
  {
    id: 'compose-mail',
    title: 'Compose New Mail',
    detail: 'Create a new email draft.',
    shortcut: 'C',
    keywords: 'compose new mail email spark draft',
    run: () => launchNewMailAuthoring(),
  },
  {
    id: 'tool-pointer',
    title: 'Switch To Pointer Tool',
    detail: 'Set the interaction tool to pointer.',
    shortcut: 'P',
    keywords: 'pointer tool annotate',
    run: () => selectInteractionTool('pointer'),
  },
  {
    id: 'tool-highlight',
    title: 'Switch To Highlight Tool',
    detail: 'Set the interaction tool to highlight.',
    shortcut: 'H',
    keywords: 'highlight tool annotate',
    run: () => selectInteractionTool('highlight'),
  },
  {
    id: 'tool-ink',
    title: 'Switch To Ink Tool',
    detail: 'Set the interaction tool to ink.',
    shortcut: 'I',
    keywords: 'ink pen tool annotate',
    run: () => selectInteractionTool('ink'),
  },
  {
    id: 'tool-text-note',
    title: 'Switch To Text Note Tool',
    detail: 'Set the interaction tool to text note.',
    shortcut: 'T',
    keywords: 'text note tool annotate',
    run: () => selectInteractionTool('text_note'),
  },
  {
    id: 'tool-prompt',
    title: 'Switch To Prompt Tool',
    detail: 'Set the interaction tool to prompt.',
    shortcut: 'Q',
    keywords: 'prompt tool annotate dictation',
    run: () => selectInteractionTool('prompt'),
  },
];

const commandCenterState = {
  query: '',
  commands: [],
  selectedIndex: 0,
};

function commandCenterRoot() {
  return document.getElementById(COMMAND_CENTER_ID);
}

function commandCenterPanel() {
  return commandCenterRoot()?.querySelector('.command-center__panel') || null;
}

function commandCenterInput() {
  return document.getElementById(COMMAND_CENTER_INPUT_ID);
}

function commandCenterList() {
  return document.getElementById(COMMAND_CENTER_LIST_ID);
}

function isEmailSidebarItem(item) {
  const artifactKind = String(item?.artifact_kind || '').trim().toLowerCase();
  return artifactKind === 'email' || artifactKind === 'email_thread';
}

function activeSidebarItem() {
  const items = Array.isArray(state.itemSidebarItems) ? state.itemSidebarItems : [];
  if (items.length === 0) return null;
  const activeID = Number(state.itemSidebarActiveItemID || 0);
  return items.find((item) => Number(item?.id || 0) === activeID) || items[0] || null;
}

function activeReplySidebarItem() {
  const item = activeSidebarItem();
  if (!item || !isEmailSidebarItem(item)) return null;
  return item;
}

function isCommandCenterVisible() {
  const root = commandCenterRoot();
  return root instanceof HTMLElement && !root.hidden;
}

function hideCommandCenter() {
  const root = commandCenterRoot();
  if (!(root instanceof HTMLElement)) return;
  root.hidden = true;
  document.body.classList.remove('command-center-open');
}

function commandMatchesQuery(command, query) {
  if (!query) return true;
  const haystack = [
    command.title,
    command.detail,
    command.keywords,
    command.shortcut,
  ].join(' ').toLowerCase();
  return query
    .split(/\s+/)
    .filter(Boolean)
    .every((token) => haystack.includes(token));
}

function availableCommandCenterCommands() {
  const commands = COMMAND_CENTER_COMMANDS.map((command) => ({ ...command }));
  const replyItem = activeReplySidebarItem();
  commands.splice(5, 0, {
    id: 'reply-mail',
    title: 'Reply To Selected Email',
    detail: replyItem
      ? `Reply to ${String(replyItem?.title || replyItem?.artifact_title || 'selected email').trim() || 'selected email'}.`
      : 'Select an email item in the sidebar to reply.',
    shortcut: 'R',
    keywords: 'reply email spark selected draft',
    disabled: !replyItem,
    run: () => (replyItem ? launchReplyAuthoring(replyItem) : false),
  });
  return commands;
}

function renderCommandCenter() {
  const list = commandCenterList();
  if (!(list instanceof HTMLElement)) return;
  const query = String(commandCenterState.query || '').trim().toLowerCase();
  const commands = availableCommandCenterCommands().filter((command) => commandMatchesQuery(command, query));
  commandCenterState.commands = commands;
  commandCenterState.selectedIndex = Math.max(0, Math.min(commandCenterState.selectedIndex, Math.max(commands.length - 1, 0)));
  list.replaceChildren();
  if (commands.length === 0) {
    const empty = document.createElement('div');
    empty.className = 'command-center__empty';
    empty.textContent = 'No commands match.';
    list.appendChild(empty);
    return;
  }
  commands.forEach((command, index) => {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'command-center__item';
    button.dataset.commandId = command.id;
    if (index === commandCenterState.selectedIndex) {
      button.classList.add('is-selected');
    }
    button.disabled = Boolean(command.disabled);

    const text = document.createElement('span');
    text.className = 'command-center__item-copy';
    const title = document.createElement('strong');
    title.textContent = command.title;
    const detail = document.createElement('span');
    detail.className = 'command-center__item-detail';
    detail.textContent = command.detail;
    text.append(title, detail);

    const shortcut = document.createElement('span');
    shortcut.className = 'command-center__shortcut';
    shortcut.textContent = command.shortcut;
    button.append(text, shortcut);
    button.addEventListener('mouseenter', () => {
      commandCenterState.selectedIndex = index;
      renderCommandCenter();
    });
    button.addEventListener('click', () => {
      commandCenterState.selectedIndex = index;
      void executeSelectedCommand();
    });
    list.appendChild(button);
  });
  const active = list.querySelector('.command-center__item.is-selected');
  if (active instanceof HTMLElement) {
    active.scrollIntoView({ block: 'nearest' });
  }
}

async function executeSelectedCommand() {
  const command = commandCenterState.commands[commandCenterState.selectedIndex] || null;
  if (!command || command.disabled) return false;
  hideCommandCenter();
  await Promise.resolve(command.run());
  return true;
}

function ensureCommandCenter() {
  const existing = commandCenterRoot();
  if (existing instanceof HTMLElement) return existing;
  const root = document.createElement('div');
  root.id = COMMAND_CENTER_ID;
  root.className = 'command-center';
  root.hidden = true;

  const panel = document.createElement('div');
  panel.className = 'command-center__panel';
  panel.setAttribute('role', 'dialog');
  panel.setAttribute('aria-modal', 'true');
  panel.setAttribute('aria-labelledby', 'command-center-title');

  const header = document.createElement('div');
  header.className = 'command-center__header';

  const titleGroup = document.createElement('div');
  const title = document.createElement('h2');
  title.id = 'command-center-title';
  title.textContent = 'Command Center';
  const hint = document.createElement('p');
  hint.textContent = 'Search commands, mail actions, and tool switches.';
  titleGroup.append(title, hint);

  const close = document.createElement('button');
  close.type = 'button';
  close.className = 'edge-btn command-center__close';
  close.textContent = 'Close';
  close.addEventListener('click', () => hideCommandCenter());
  header.append(titleGroup, close);

  const input = document.createElement('input');
  input.id = COMMAND_CENTER_INPUT_ID;
  input.className = 'command-center__input';
  input.type = 'text';
  input.autocomplete = 'off';
  input.placeholder = 'Type to filter commands';
  input.setAttribute('aria-label', 'Filter commands');
  input.addEventListener('input', () => {
    commandCenterState.query = String(input.value || '');
    commandCenterState.selectedIndex = 0;
    renderCommandCenter();
  });

  const list = document.createElement('div');
  list.id = COMMAND_CENTER_LIST_ID;
  list.className = 'command-center__list';

  panel.append(header, input, list);
  root.appendChild(panel);
  root.addEventListener('mousedown', (event) => {
    if (event.target === root) {
      hideCommandCenter();
    }
  });
  document.body.appendChild(root);
  return root;
}

function openCommandCenter() {
  hideTextInput();
  hideOverlay();
  cancelLiveSessionListen();
  const root = ensureCommandCenter();
  root.hidden = false;
  document.body.classList.add('command-center-open');
  commandCenterState.query = '';
  commandCenterState.selectedIndex = 0;
  const input = commandCenterInput();
  if (input instanceof HTMLInputElement) {
    input.value = '';
    input.focus();
    input.select();
  }
  renderCommandCenter();
}

function moveCommandCenterSelection(delta) {
  if (commandCenterState.commands.length === 0) return;
  const count = commandCenterState.commands.length;
  commandCenterState.selectedIndex = (commandCenterState.selectedIndex + delta + count) % count;
  renderCommandCenter();
}

function handleCommandCenterShortcut(ev) {
  const key = String(ev.key || '').toLowerCase();
  if ((ev.metaKey || ev.ctrlKey) && !ev.altKey && key === 'k') {
    ev.preventDefault();
    ev.stopPropagation();
    if (isCommandCenterVisible()) {
      hideCommandCenter();
    } else {
      openCommandCenter();
    }
    return true;
  }
  if (!isCommandCenterVisible()) return false;
  if (ev.key === 'Escape' && !ev.metaKey && !ev.ctrlKey && !ev.altKey) {
    ev.preventDefault();
    hideCommandCenter();
    return true;
  }
  if (ev.key === 'ArrowDown') {
    ev.preventDefault();
    moveCommandCenterSelection(1);
    return true;
  }
  if (ev.key === 'ArrowUp') {
    ev.preventDefault();
    moveCommandCenterSelection(-1);
    return true;
  }
  if (ev.key === 'Enter') {
    ev.preventDefault();
    void executeSelectedCommand();
    return true;
  }
  return false;
}

function handleMailShortcut(ev) {
  if (ev.metaKey || ev.ctrlKey || ev.altKey) return false;
  if (state.prReviewMode || state.fileSidebarMode !== 'items') return false;
  if (!document.body.classList.contains('file-sidebar-open')) return false;
  if (String(ev.key || '') === 'c' || String(ev.key || '') === 'C') {
    ev.preventDefault();
    void launchNewMailAuthoring();
    return true;
  }
  if (String(ev.key || '') !== 'r' && String(ev.key || '') !== 'R') return false;
  const replyItem = activeReplySidebarItem();
  if (!replyItem) return false;
  ev.preventDefault();
  void launchReplyAuthoring(replyItem);
  return true;
}

export function bindUi() {
  const canvasText = document.getElementById('canvas-text');
  const canvasViewport = document.getElementById('canvas-viewport');
  const artifactEditor = ensureArtifactEditor();
  const indicatorNode = document.getElementById('indicator');
  if (indicatorNode && indicatorNode.parentElement !== document.body) {
    document.body.appendChild(indicatorNode);
  }
  ensureCommandCenter();
  if (artifactEditor) {
    artifactEditor.addEventListener('keydown', (ev) => {
      if (ev.key !== 'Escape') return;
      ev.preventDefault();
      ev.stopPropagation();
      exitArtifactEditMode({ applyChanges: true });
    }, true);
  }
  let lastMouseX = Math.floor(window.innerWidth / 2);
  let lastMouseY = Math.floor(window.innerHeight / 2);
  let hasLastMousePosition = false;
  const isInEdgeZone = (x, y) => {
    const s = getEdgeTapSizePx();
    const top = getTopEdgeTapSizePx();
    return x < s || x > window.innerWidth - s || y < top || y > window.innerHeight - s;
  };
  const isVoiceInteractionTarget = (target, x, y) => (
    isInEdgeZone(x, y)
    || (target instanceof Element
      && target.closest('button,a,input,textarea,select,[contenteditable="true"],.overlay,.floating-input,.edge-panel,#canvas-pdf .canvas-pdf-page,#canvas-pdf .textLayer,#canvas-pdf .annotationLayer'))
  );
  const rememberMousePosition = (x, y) => {
    if (!Number.isFinite(x) || !Number.isFinite(y)) return;
    lastMouseX = Number(x);
    lastMouseY = Number(y);
    hasLastMousePosition = true;
  };
  const getCtrlVoiceCapturePoint = () => {
    if (hasLastMousePosition) {
      return { x: lastMouseX, y: lastMouseY };
    }
    const lastPos = getLastInputPosition();
    if (Number.isFinite(lastPos?.x) && Number.isFinite(lastPos?.y)) {
      return { x: Number(lastPos.x), y: Number(lastPos.y) };
    }
    return {
      x: Math.floor(window.innerWidth / 2),
      y: Math.floor(window.innerHeight / 2),
    };
  };
  const beginVoiceCaptureFromPoint = (x, y) => {
    let anchor = null;
    if (state.hasArtifact && canvasText) {
      anchor = getAnchorFromPoint(x, y);
    }
    return beginVoiceCapture(x, y, anchor);
  };
  const buildCanvasPositionPayload = (anchor, options = {}) => {
    if (!anchor || typeof anchor !== 'object') return null;
    const payload = {
      type: 'canvas_position',
      gesture: String(options?.gesture || 'tap').trim().toLowerCase() || 'tap',
      output_mode: state.ttsSilent ? 'silent' : 'voice',
      cursor: {},
    };
    const cursor = payload.cursor;
    const setTextField = (key, value) => {
      const text = String(value || '').trim();
      if (text) cursor[key] = text;
    };
    const setIntField = (key, value) => {
      const num = Number.parseInt(String(value || ''), 10);
      if (Number.isFinite(num) && num > 0) cursor[key] = num;
    };
    const setFloatField = (key, value) => {
      const num = Number(value);
      if (Number.isFinite(num)) cursor[key] = num;
    };
    setTextField('view', anchor.view);
    setTextField('element', anchor.element);
    setTextField('title', anchor.title);
    setIntField('page', anchor.page);
    setIntField('line', anchor.line);
    setFloatField('relative_x', anchor.relativeX);
    setFloatField('relative_y', anchor.relativeY);
    setTextField('selected_text', anchor.selectedText);
    setTextField('surrounding_text', anchor.surroundingText);
    setIntField('item_id', anchor.itemID || anchor.item_id);
    setTextField('item_title', anchor.itemTitle || anchor.item_title);
    setTextField('item_state', anchor.itemState || anchor.item_state);
    setIntField('workspace_id', anchor.workspaceID || anchor.workspace_id);
    setTextField('workspace_name', anchor.workspaceName || anchor.workspace_name);
    setTextField('path', anchor.path);
    if (anchor.isDir === true || anchor.is_dir === true) {
      cursor.is_dir = true;
    }
    if (options?.requestResponse) {
      payload.request_response = true;
    }
    if (Object.keys(cursor).length === 0) return null;
    return payload;
  };
  const sendCanvasPosition = (anchor, options = {}) => {
    const payload = buildCanvasPositionPayload(anchor, options);
    const ws = state.chatWs;
    if (!payload || !ws || ws.readyState !== WebSocket.OPEN) return false;
    ws.send(JSON.stringify(payload));
    return true;
  };

  document.addEventListener('mousemove', (ev) => {
    rememberMousePosition(ev.clientX, ev.clientY);
  }, { passive: true });
  document.addEventListener('pointerdown', (ev) => {
    if (ev.pointerType !== 'mouse') return;
    rememberMousePosition(ev.clientX, ev.clientY);
  }, true);

  if (indicatorNode) {
    const isIndicatorArmed = () => (
      indicatorNode.classList.contains('is-working')
      || indicatorNode.classList.contains('is-recording')
      || indicatorNode.classList.contains('is-listening')
    );
    const pointHitsIndicatorChip = (x, y) => {
      const chips = indicatorNode.querySelectorAll('.record-dot, .stop-square');
      for (const chip of chips) {
        if (!(chip instanceof HTMLElement)) continue;
        const style = window.getComputedStyle(chip);
        if (style.display === 'none' || style.visibility === 'hidden') continue;
        const rect = chip.getBoundingClientRect();
        if (x >= rect.left && x <= rect.right && y >= rect.top && y <= rect.bottom) {
          return true;
        }
      }
      return false;
    };
    const isTapOnInteractiveUi = (ev) => {
      const t = ev.target;
      if (!(t instanceof Element)) return false;
      return Boolean(t.closest('button, a, input, textarea, select, #edge-left-tap, #edge-right-tap, #edge-top, #edge-right, #pr-file-pane, #pr-file-drawer-backdrop'));
    };
    const handleIndicatorTap = (ev, x, y, isTouch = false) => {
      if (!isIndicatorArmed()) return;
      if (!isTouch && isSuppressedClick()) return;
      const stopGestureActive = isUiStopGestureActive();
      const hitsChip = pointHitsIndicatorChip(x, y);
      if (!hitsChip && isTouch && stopGestureActive && isTapOnInteractiveUi(ev)) return;
      if (!hitsChip && !(isTouch && stopGestureActive)) return;
      ev.preventDefault();
      ev.stopPropagation();
      if (isTouch) suppressSyntheticClick();
      void handleStopAction();
    };
    document.addEventListener('click', (ev) => {
      handleIndicatorTap(ev, ev.clientX, ev.clientY, false);
    }, true);
    document.addEventListener('touchend', (ev) => {
      const touch = ev.changedTouches && ev.changedTouches.length > 0 ? ev.changedTouches[0] : null;
      if (!touch) return;
      handleIndicatorTap(ev, touch.clientX, touch.clientY, true);
    }, { passive: false, capture: true });
  }

  // Left-click/tap on canvas -> toggle voice recording
  const clickTarget = canvasViewport || document.getElementById('workspace');
  const syncIndicatorOnViewportChange = () => {
    updateAssistantActivityIndicator();
  };
  if (canvasViewport instanceof HTMLElement) {
    syncInkLayerSize();
    canvasViewport.addEventListener('scroll', syncIndicatorOnViewportChange, { passive: true, capture: true });
    let canvasSwipeStart = null;
    let canvasSwipeHandled = false;
    let horizontalWheelAccum = 0;
    let horizontalWheelLastAt = 0;
    const resetCanvasSwipe = () => {
      canvasSwipeStart = null;
      canvasSwipeHandled = false;
    };
    canvasViewport.addEventListener('touchstart', (ev) => {
      if (!isMobileViewport() && !isLikelyIOS()) return;
      if (state.prReviewDrawerOpen || ev.touches.length !== 1) return;
      const touch = ev.touches[0];
      canvasSwipeStart = { x: touch.clientX, y: touch.clientY };
      canvasSwipeHandled = false;
    }, { passive: true });
    canvasViewport.addEventListener('touchmove', (ev) => {
      if (!canvasSwipeStart || canvasSwipeHandled || ev.touches.length !== 1) return;
      const touch = ev.touches[0];
      const dx = touch.clientX - canvasSwipeStart.x;
      const dy = touch.clientY - canvasSwipeStart.y;
      if (!state.hasArtifact) return;
      if (Math.abs(dx) < 48) return;
      if (Math.abs(dx) <= Math.abs(dy) * 1.25) return;
      const stepped = stepCanvasFile(dx < 0 ? 1 : -1);
      if (!stepped) return;
      canvasSwipeHandled = true;
      ev.preventDefault();
    }, { passive: false });
    canvasViewport.addEventListener('touchend', resetCanvasSwipe, { passive: true });
    canvasViewport.addEventListener('touchcancel', resetCanvasSwipe, { passive: true });
    canvasViewport.addEventListener('wheel', (ev) => {
      if (!state.hasArtifact) return;
      const absX = Math.abs(ev.deltaX);
      const absY = Math.abs(ev.deltaY);
      if (absX < 0.8) return;
      if (absX <= absY * 1.15) return;
      ev.preventDefault();
      const now = Date.now();
      if (now - horizontalWheelLastAt > 260) {
        horizontalWheelAccum = 0;
      }
      horizontalWheelAccum += ev.deltaX;
      if (Math.abs(horizontalWheelAccum) < 48) return;
      const stepped = stepCanvasFile(horizontalWheelAccum > 0 ? 1 : -1);
      if (!stepped) return;
      horizontalWheelAccum = 0;
      horizontalWheelLastAt = now;
    }, { passive: false });
    canvasViewport.addEventListener('pointerdown', (ev) => {
      if (!isInkTool()) return;
      if (ev.pointerType !== 'pen') return;
      if (isEditableTarget(ev.target)) return;
      if (ev.target instanceof Element && ev.target.closest('.edge-panel,#pr-file-pane,#pr-file-drawer-backdrop')) return;
      if (beginInkStroke(ev)) {
        try { window.getSelection()?.removeAllRanges(); } catch (_) {}
        setPenInkingState(true);
        ev.preventDefault();
        try { canvasViewport.setPointerCapture(ev.pointerId); } catch (_) {}
      }
    }, true);
    canvasViewport.addEventListener('pointermove', (ev) => {
      if (!isInkTool()) return;
      if (state.inkDraft.activePointerId !== ev.pointerId) return;
      if (extendInkStroke(ev)) {
        ev.preventDefault();
      }
    }, true);
    const finishInkPointer = (ev) => {
      if (state.inkDraft.activePointerId !== ev.pointerId) return;
      if (!finalizeInkStroke(ev)) {
        extendInkStroke(ev);
        resetInkDraftState();
      }
      setPenInkingState(false);
      ev.preventDefault();
    };
    canvasViewport.addEventListener('pointerup', finishInkPointer, true);
    canvasViewport.addEventListener('pointercancel', finishInkPointer, true);
    canvasViewport.addEventListener('selectstart', (ev) => {
      if (!isInkTool()) return;
      ev.preventDefault();
    }, true);
  }
  window.addEventListener('scroll', syncIndicatorOnViewportChange, { passive: true });
  window.addEventListener('resize', syncIndicatorOnViewportChange);

  if (clickTarget) {
    let touchTapStartX = 0;
    let touchTapStartY = 0;
    let touchTapTracking = false;
    let touchTapMoved = false;
    let touchLongTapTriggered = false;
    let touchEditTimer = null;
    const TOUCH_TAP_MOVE_THRESHOLD = 10;
    const clearTouchEditTimer = () => {
      if (touchEditTimer !== null) {
        clearTimeout(touchEditTimer);
        touchEditTimer = null;
      }
    };

    const handleWorkspaceTap = (target, x, y) => {
      const requestedPositionPrompt = String(state.requestedPositionPrompt || '').trim();
      if (requestedPositionPrompt) {
        if (isVoiceInteractionTarget(target, x, y)) return;
        const sel = window.getSelection();
        if (sel && !sel.isCollapsed) return;
        rememberMousePosition(x, y);
        const anchor = state.hasArtifact && canvasText ? getAnchorFromPoint(x, y) : null;
        pinCursorAnchor(x, y, anchor);
        state.requestedPositionPrompt = '';
        sendCanvasPosition(anchor, { gesture: 'tap', requestResponse: true });
        showStatus('position shared');
        updateAssistantActivityIndicator();
        return;
      }
      const liveSessionPointerMode = state.liveSessionActive
        && (state.liveSessionMode === LIVE_SESSION_MODE_DIALOGUE || state.liveSessionMode === LIVE_SESSION_MODE_MEETING);
      if (liveSessionPointerMode) {
        if (prefersTextComposer() && state.hasArtifact && createPdfStickyNoteAt(x, y)) return;
        if (isVoiceInteractionTarget(target, x, y)) return;
        const sel = window.getSelection();
        if (sel && !sel.isCollapsed) return;
        rememberMousePosition(x, y);
        const anchor = state.hasArtifact && canvasText ? getAnchorFromPoint(x, y) : null;
        pinCursorAnchor(x, y, anchor);
        sendCanvasPosition(anchor, { gesture: 'tap' });
        updateAssistantActivityIndicator();
        return;
      }
      if (isLiveSessionListenActive()) {
        if (isVoiceInteractionTarget(target, x, y)) return;
        cancelLiveSessionListen();
        if (prefersTextComposer()) {
          const anchor = state.hasArtifact && canvasText ? getAnchorFromPoint(x, y) : null;
          openComposerAt(x, y, anchor);
        } else {
          void beginVoiceCaptureFromPoint(x, y);
        }
        return;
      }
      if (isUiStopGestureActive()) {
        void handleStopAction();
        return;
      }
      if (prefersTextComposer() && state.hasArtifact && createPdfStickyNoteAt(x, y)) return;
      if (isVoiceInteractionTarget(target, x, y)) return;
      const sel = window.getSelection();
      if (sel && !sel.isCollapsed) return;
      rememberMousePosition(x, y);
      if (isRecording()) {
        void stopVoiceCaptureAndSend();
        return;
      }
      if (prefersTextComposer()) {
        const anchor = state.hasArtifact && canvasText ? getAnchorFromPoint(x, y) : null;
        openComposerAt(x, y, anchor);
        return;
      }
      if (state.interaction.conversation === 'push_to_talk') {
        void beginVoiceCaptureFromPoint(x, y);
      }
    };

    clickTarget.addEventListener('touchstart', (ev) => {
      if (ev.touches.length !== 1) {
        touchTapTracking = false;
        touchTapMoved = false;
        touchLongTapTriggered = false;
        clearTouchEditTimer();
        return;
      }
      const touch = ev.touches[0];
      if (isEditableTarget(ev.target)) {
        touchTapTracking = false;
        touchTapMoved = false;
        touchLongTapTriggered = false;
        clearTouchEditTimer();
        return;
      }
      touchTapStartX = touch.clientX;
      touchTapStartY = touch.clientY;
      touchTapTracking = !isVoiceInteractionTarget(ev.target, touch.clientX, touch.clientY);
      touchTapMoved = false;
      touchLongTapTriggered = false;
      clearTouchEditTimer();
      if (touchTapTracking && canEnterArtifactEditModeFromTarget(ev.target)) {
        touchEditTimer = window.setTimeout(() => {
          touchEditTimer = null;
          touchTapTracking = false;
          touchTapMoved = false;
          touchLongTapTriggered = enterArtifactEditMode(touchTapStartX, touchTapStartY);
          if (touchLongTapTriggered) suppressSyntheticClick();
        }, ARTIFACT_EDIT_LONG_TAP_MS);
      }
    }, { passive: true });

    clickTarget.addEventListener('touchmove', (ev) => {
      if ((!touchTapTracking && touchEditTimer === null) || touchTapMoved || ev.touches.length !== 1) return;
      const touch = ev.touches[0];
      if (Math.hypot(touch.clientX - touchTapStartX, touch.clientY - touchTapStartY) > TOUCH_TAP_MOVE_THRESHOLD) {
        touchTapMoved = true;
        clearTouchEditTimer();
      }
    }, { passive: true });

    clickTarget.addEventListener('touchend', (ev) => {
      if (touchLongTapTriggered) {
        touchLongTapTriggered = false;
        touchTapTracking = false;
        touchTapMoved = false;
        clearTouchEditTimer();
        ev.preventDefault();
        suppressSyntheticClick();
        return;
      }
      if (!touchTapTracking) return;
      touchTapTracking = false;
      if (touchTapMoved) {
        touchTapMoved = false;
        clearTouchEditTimer();
        return;
      }
      const touch = ev.changedTouches && ev.changedTouches.length > 0 ? ev.changedTouches[0] : null;
      if (!touch) return;
      clearTouchEditTimer();
      ev.preventDefault();
      suppressSyntheticClick();
      handleWorkspaceTap(ev.target, touch.clientX, touch.clientY);
    }, { passive: false });

    clickTarget.addEventListener('touchcancel', () => {
      touchTapTracking = false;
      touchTapMoved = false;
      touchLongTapTriggered = false;
      clearTouchEditTimer();
    }, { passive: true });

    clickTarget.addEventListener('click', (ev) => {
      if (isSuppressedClick()) return;
      if (ev.button !== 0) return;
      handleWorkspaceTap(ev.target, ev.clientX, ev.clientY);
    });
  }

  // Right-click -> artifact editor (text artifacts) or floating text input
  if (clickTarget) {
    clickTarget.addEventListener('contextmenu', (ev) => {
      if (state.artifactEditMode) {
        ev.preventDefault();
        return;
      }
      if (ev.target instanceof Element && ev.target.closest('.edge-panel')) return;
      if (canEnterArtifactEditModeFromTarget(ev.target)) {
        ev.preventDefault();
        enterArtifactEditMode(ev.clientX, ev.clientY);
        return;
      }
      ev.preventDefault();
      cancelLiveSessionListen();
      let anchor = null;
      if (state.hasArtifact && canvasText) {
        anchor = getAnchorFromPoint(ev.clientX, ev.clientY);
      }
      openComposerAt(ev.clientX, ev.clientY, anchor);
    });
  }

  // Text input Enter -> send
  const floatingInput = document.getElementById('floating-input');
  if (floatingInput instanceof HTMLTextAreaElement) {
    floatingInput.addEventListener('focus', () => {
      cancelLiveSessionListen();
    });
    floatingInput.addEventListener('keydown', (ev) => {
      if (ev.key === 'Enter' && !ev.shiftKey) {
        ev.preventDefault();
        const text = floatingInput.value.trim();
        if (text) {
          state.lastInputOrigin = 'text';
          floatingInput.value = '';
          floatingInput.blur();
          hideTextInput();
          settleKeyboardAfterSubmit();
          void submitMessage(text);
        }
      }
      if (ev.key === 'Escape') {
        ev.preventDefault();
        hideTextInput();
      }
    });
    floatingInput.addEventListener('input', () => {
      floatingInput.style.height = 'auto';
      floatingInput.style.height = `${Math.min(floatingInput.scrollHeight, 240)}px`;
    });
  }

  // Chat pane input: Enter sends, Escape blurs, auto-resize
  const chatPaneInput = document.getElementById('chat-pane-input');
  if (chatPaneInput instanceof HTMLTextAreaElement) {
    chatPaneInput.addEventListener('focus', () => {
      cancelLiveSessionListen();
    });
    chatPaneInput.addEventListener('keydown', (ev) => {
      if (ev.key === 'Enter' && !ev.shiftKey) {
        ev.preventDefault();
        const text = chatPaneInput.value.trim();
        if (text) {
          state.lastInputOrigin = 'text';
          chatPaneInput.value = '';
          chatPaneInput.style.height = '';
          chatPaneInput.blur();
          settleKeyboardAfterSubmit();
          void submitMessage(text);
        }
      }
      if (ev.key === 'Escape') {
        ev.preventDefault();
        chatPaneInput.value = '';
        chatPaneInput.style.height = '';
        chatPaneInput.blur();
        settleKeyboardAfterSubmit();
      }
    });
    chatPaneInput.addEventListener('input', () => {
      chatPaneInput.style.height = 'auto';
      chatPaneInput.style.height = `${Math.min(chatPaneInput.scrollHeight, 240)}px`;
    });

  }

  const inkClear = document.getElementById('ink-clear');
  if (inkClear instanceof HTMLButtonElement) {
    inkClear.addEventListener('click', () => {
      clearInkDraft();
      showStatus('ink cleared');
    });
  }
  const inkSubmit = document.getElementById('ink-submit');
  if (inkSubmit instanceof HTMLButtonElement) {
    inkSubmit.addEventListener('click', () => {
      void submitInkDraft();
    });
  }

  // Voice tap on chat history (only when panel is pinned, not just hover-active)
  const chatHistory = document.getElementById('chat-history');
  if (chatHistory) {
    chatHistory.addEventListener('click', (ev) => {
      if (ev.button !== 0) return;
      if (ev.target instanceof Element && ev.target.closest('a,button,input,textarea,select,[contenteditable="true"]')) return;
      if (isInEdgeZone(ev.clientX, ev.clientY)) return;
      const edgeR = chatHistory.closest('.edge-panel');
      if (edgeR && !edgeR.classList.contains('edge-pinned')) return;
      if (isUiStopGestureActive()) { void handleStopAction(); return; }
      if (prefersTextComposer()) return;
      if (isLiveSessionListenActive()) {
        cancelLiveSessionListen();
        void beginVoiceCaptureFromPoint(ev.clientX, ev.clientY);
        return;
      }
      if (isRecording()) { void stopVoiceCaptureAndSend(); return; }
      void beginVoiceCaptureFromPoint(ev.clientX, ev.clientY);
    });
  }

  // Click outside overlay/input -> dismiss
  document.addEventListener('mousedown', (ev) => {
    if (!(ev.target instanceof Element)) return;
    const sidebarMenu = document.getElementById(ITEM_SIDEBAR_MENU_ID);
    if (state.itemSidebarMenuOpen && sidebarMenu instanceof HTMLElement && !sidebarMenu.contains(ev.target)) {
      hideItemSidebarMenu();
    }
    const commandCenter = commandCenterPanel();
    if (isCommandCenterVisible() && commandCenter instanceof HTMLElement && !commandCenter.contains(ev.target)) {
      hideCommandCenter();
    }
    // Dismiss overlay on click outside
    if (isOverlayVisible()) {
      const overlay = document.getElementById('overlay');
      if (overlay && !overlay.contains(ev.target)) {
        hideOverlay();
      }
    }
    // Dismiss text input on click outside
    if (isTextInputVisible()) {
      const input = document.getElementById('floating-input');
      if (input && !input.contains(ev.target) && ev.button === 0) {
        hideTextInput();
      }
    }
  });

  // Keyboard typing auto-activates text input (rasa mode)
  document.addEventListener('keydown', (ev) => {
    if (handleCommandCenterShortcut(ev)) {
      return;
    }
    // Escape handling
    if (ev.key === 'Escape' && !ev.metaKey && !ev.ctrlKey && !ev.altKey) {
      if (state.artifactEditMode) {
        ev.preventDefault();
        exitArtifactEditMode({ applyChanges: true });
        return;
      }
      if (isRecording()) {
        cancelChatVoiceCapture();
        showStatus('ready');
        return;
      }
      if (isOverlayVisible()) {
        hideOverlay();
        return;
      }
      if (isTextInputVisible()) {
        hideTextInput();
        return;
      }
      if (state.itemSidebarMenuOpen) {
        hideItemSidebarMenu();
        return;
      }
      if (state.inkDraft.dirty) {
        clearInkDraft();
        showStatus('ink cleared');
        return;
      }
      if (state.prReviewDrawerOpen) {
        setPrReviewDrawerOpen(false);
        return;
      }
      closeEdgePanels();
      if (state.hasArtifact) {
        clearCanvas();
        hideCanvasColumn();
        return;
      }
      void handleStopAction();
      return;
    }

    // Enter stops recording
    if (ev.key === 'Enter' && isRecording()) {
      ev.preventDefault();
      void stopVoiceCaptureAndSend();
      return;
    }
    if (ev.key === 'Enter' && isInkTool() && state.inkDraft.dirty) {
      ev.preventDefault();
      void submitInkDraft();
      return;
    }

    // Control long-press for PTT
    if (ev.key === 'Control' && !ev.repeat) {
      if (state.chatCtrlHoldTimer || state.chatVoiceCapture) return;
      if (isLiveSessionListenActive()) {
        cancelLiveSessionListen();
      }
      state.chatCtrlHoldTimer = window.setTimeout(() => {
        state.chatCtrlHoldTimer = null;
        const point = getCtrlVoiceCapturePoint();
        void beginVoiceCaptureFromPoint(point.x, point.y);
      }, CHAT_CTRL_LONG_PRESS_MS);
      return;
    }

    if (ev.ctrlKey && ev.key !== 'Control') {
      if (state.chatCtrlHoldTimer) {
        clearTimeout(state.chatCtrlHoldTimer);
        state.chatCtrlHoldTimer = null;
      }
      if (state.chatVoiceCapture) {
        cancelChatVoiceCapture();
        showStatus('ready');
      }
      return;
    }

    if (isCommandCenterVisible()) return;
    if (ev.metaKey || ev.ctrlKey || ev.altKey) return;
    if (isEditableTarget(ev.target)) return;
    if (state.artifactEditMode) return;
    if (handleItemSidebarKeyboardShortcut(ev)) return;
    if (handleMailShortcut(ev)) return;

    if (ev.key === 'ArrowRight') {
      if (stepCanvasFile(1)) {
        ev.preventDefault();
      }
      return;
    }
    if (ev.key === 'ArrowLeft') {
      if (stepCanvasFile(-1)) {
        ev.preventDefault();
      }
      return;
    }

    if (state.prReviewMode) {
      if (ev.key === 'j' || ev.key === 'J') {
        ev.preventDefault();
        stepPrReviewFile(1);
        return;
      }
      if (ev.key === 'k' || ev.key === 'K') {
        ev.preventDefault();
        stepPrReviewFile(-1);
        return;
      }
    }

    // Route printable keys into an active composer before treating them as tool shortcuts.
    if (ev.key.length === 1 && !isTextInputVisible()) {
      const edgeR = document.getElementById('edge-right');
      const cpInput = document.getElementById('chat-pane-input');
      const chatPaneOpen = edgeR && (edgeR.classList.contains('edge-active') || edgeR.classList.contains('edge-pinned'));
      if (chatPaneOpen && cpInput instanceof HTMLTextAreaElement && !window.matchMedia('(max-width: 767px)').matches) {
        cancelLiveSessionListen();
        cpInput.focus();
        cpInput.value = ev.key;
        const caret = ev.key.length;
        cpInput.setSelectionRange(caret, caret);
        cpInput.dispatchEvent(new Event('input', { bubbles: true }));
        ev.preventDefault();
        return;
      }
      if (prefersTextComposer()) {
        const cx = window.innerWidth / 2 - 130;
        const cy = window.innerHeight / 2;
        cancelLiveSessionListen();
        openComposerAt(cx, cy, null, ev.key);
        ev.preventDefault();
        return;
      }
    }

    const toolByKey = {
      '1': 'pointer',
      '2': 'highlight',
      '3': 'ink',
      '4': 'text_note',
      '5': 'prompt',
      p: 'pointer',
      P: 'pointer',
      h: 'highlight',
      H: 'highlight',
      i: 'ink',
      I: 'ink',
      t: 'text_note',
      T: 'text_note',
      q: 'prompt',
      Q: 'prompt',
    };
    const shortcutTool = toolByKey[ev.key];
    if (shortcutTool) {
      ev.preventDefault();
      void selectInteractionTool(shortcutTool);
      return;
    }

    // Enter when text input is NOT visible but could send
    if (ev.key === 'Enter' && !isTextInputVisible()) {
      ev.preventDefault();
    }
  }, true);

  document.addEventListener('keyup', (ev) => {
    if (ev.key !== 'Control') return;
    if (state.chatCtrlHoldTimer) {
      clearTimeout(state.chatCtrlHoldTimer);
      state.chatCtrlHoldTimer = null;
      return;
    }
    if (state.chatVoiceCapture) {
      void stopVoiceCaptureAndSend();
    }
  }, true);

  window.addEventListener('blur', () => {
    if (state.chatCtrlHoldTimer) {
      clearTimeout(state.chatCtrlHoldTimer);
      state.chatCtrlHoldTimer = null;
    }
    // Keep active capture alive on transient browser blur; hard stop is
    // handled by visibilitychange when the page is actually hidden.
    if (state.chatVoiceCapture && document.hidden) {
      cancelChatVoiceCapture();
      showStatus('ready');
    }
  });

  // Text selection on artifact sets anchor
  if (canvasText) {
    canvasText.addEventListener('mouseup', () => {
      const sel = window.getSelection();
      if (!sel || sel.isCollapsed) return;
      const loc = getLocationFromSelection();
      if (loc) {
        setInputAnchor({ line: loc.line, title: loc.title, selectedText: loc.selectedText });
      }
    });
  }
  const applySelectionHighlightSoon = () => {
    window.setTimeout(() => {
      maybeApplySelectionHighlight();
    }, 0);
  };
  document.addEventListener('mouseup', applySelectionHighlightSoon, true);
  document.addEventListener('touchend', applySelectionHighlightSoon, true);

  initEdgePanels();
}
