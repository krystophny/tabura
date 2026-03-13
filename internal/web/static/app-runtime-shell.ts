import * as context from './app-context.js';

const { PANEL_MOTION_WATCH_QUERIES } = context;

let panelMotionWatchersAttached = false;
let syncKeyboardStateNow = null;

export function mediaQueryMatches(query) {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return false;
  try {
    return window.matchMedia(query).matches;
  } catch (_) {
    return false;
  }
}

export function shouldEnablePanelMotion() {
  if (mediaQueryMatches('(prefers-reduced-motion: reduce)')) return false;
  if (mediaQueryMatches('(monochrome)')) return false;
  if (mediaQueryMatches('(update: slow)')) return false;
  return true;
}

export function syncPanelMotionMode() {
  document.body.classList.toggle('panel-motion-enabled', shouldEnablePanelMotion());
}

export function initPanelMotionMode() {
  syncPanelMotionMode();
  if (panelMotionWatchersAttached) return;
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return;
  panelMotionWatchersAttached = true;
  PANEL_MOTION_WATCH_QUERIES.forEach((query) => {
    let mql = null;
    try {
      mql = window.matchMedia(query);
    } catch (_) {
      mql = null;
    }
    if (!mql) return;
    const onChange = () => syncPanelMotionMode();
    if (typeof mql.addEventListener === 'function') {
      mql.addEventListener('change', onChange);
      return;
    }
    if (typeof mql.addListener === 'function') {
      mql.addListener(onChange);
    }
  });
}

const IPHONE_CORNER_RADIUS_PROFILES = [
  { shortSide: 375, longSide: 812, dpr: 3, radius: 44 },
  { shortSide: 390, longSide: 844, dpr: 3, radius: 47 },
  { shortSide: 393, longSide: 852, dpr: 3, radius: 55 },
  { shortSide: 402, longSide: 874, dpr: 3, radius: 62 },
  { shortSide: 414, longSide: 896, dpr: 2, radius: 41 },
  { shortSide: 428, longSide: 926, dpr: 3, radius: 53 },
  { shortSide: 430, longSide: 932, dpr: 3, radius: 55 },
  { shortSide: 440, longSide: 956, dpr: 3, radius: 62 },
];

export function isIPhoneStandalone() {
  const ua = String(navigator.userAgent || '').toLowerCase();
  const plat = String(navigator.platform || '').toLowerCase();
  const isIPhone = /iphone/.test(ua) || plat === 'iphone' || (plat === 'macintel' && navigator.maxTouchPoints > 1);
  if (!isIPhone) return false;
  try {
    return (navigator as any).standalone === true || window.matchMedia('(display-mode: standalone)').matches;
  } catch (_) {
    return false;
  }
}

export function applyIPhoneFrameCorners() {
  const root = document.documentElement;
  if (!isIPhoneStandalone()) {
    root.style.removeProperty('--cue-corner-radius');
    return;
  }
  const short = Math.min(Math.round(screen.width), Math.round(screen.height));
  const long = Math.max(Math.round(screen.width), Math.round(screen.height));
  const dpr = Math.max(1, Math.round(devicePixelRatio || 1));
  const match = IPHONE_CORNER_RADIUS_PROFILES.find(
    (p) => p.shortSide === short && p.longSide === long && p.dpr === dpr,
  );
  const r = match ? match.radius : (dpr >= 3 ? 55 : 44);
  root.style.setProperty('--cue-corner-radius', `0 0 ${r}px ${r}px`);
}

export function setSyncKeyboardStateNow(sync) {
  syncKeyboardStateNow = typeof sync === 'function' ? sync : null;
}

export function isFocusedTextInput() {
  const el = document.activeElement;
  if (!el) return false;
  if (el instanceof HTMLTextAreaElement) return true;
  if (el instanceof HTMLInputElement) {
    const type = String(el.type || 'text').toLowerCase();
    return ![
      'button', 'checkbox', 'color', 'file', 'hidden',
      'image', 'radio', 'range', 'reset', 'submit',
    ].includes(type);
  }
  return el instanceof HTMLElement && el.isContentEditable;
}

export function clearKeyboardOpenState() {
  const inputRow = document.querySelector('.chat-pane-input-row');
  if (inputRow) inputRow.classList.remove('keyboard-open');
  document.body.classList.remove('keyboard-open');
  if (isIPhoneStandalone()) applyIPhoneFrameCorners();
}

export function settleKeyboardAfterSubmit() {
  clearKeyboardOpenState();
  const sync = syncKeyboardStateNow;
  if (typeof sync !== 'function') return;
  [0, 100, 220, 380, 600, 900, 1300].forEach((delay) => {
    window.setTimeout(() => {
      if (syncKeyboardStateNow !== sync) return;
      sync();
    }, delay);
  });
}
