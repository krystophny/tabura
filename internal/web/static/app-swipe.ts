import * as context from './app-context.js';

const {
  HORIZONTAL_GESTURE_MOVE_PX,
  HORIZONTAL_SWIPE_COMMIT_PX,
  HORIZONTAL_SWIPE_AXIS_RATIO,
  HORIZONTAL_SWIPE_HOLD_MS,
} = context;

export type HorizontalSwipeState = {
  startX: number,
  startY: number,
  lastX: number,
  lastY: number,
  startedAt: number,
  held: boolean,
};

export function beginHorizontalSwipe(touch: Pick<Touch, 'clientX' | 'clientY'>): HorizontalSwipeState {
  return {
    startX: Number(touch.clientX) || 0,
    startY: Number(touch.clientY) || 0,
    lastX: Number(touch.clientX) || 0,
    lastY: Number(touch.clientY) || 0,
    startedAt: Date.now(),
    held: false,
  };
}

export function horizontalSwipeDelta(
  state: HorizontalSwipeState | null,
  touch: Pick<Touch, 'clientX' | 'clientY'> | null,
) {
  if (!state || !touch) {
    return { dx: 0, dy: 0 };
  }
  const dx = (Number(touch.clientX) || 0) - state.startX;
  const dy = (Number(touch.clientY) || 0) - state.startY;
  state.lastX = Number(touch.clientX) || 0;
  state.lastY = Number(touch.clientY) || 0;
  return { dx, dy };
}

export function hasGestureMoved(dx: number, dy: number, threshold = HORIZONTAL_GESTURE_MOVE_PX) {
  return Math.hypot(Number(dx) || 0, Number(dy) || 0) > threshold;
}

export function isHorizontalSwipeIntent(
  dx: number,
  dy: number,
  commitPx = HORIZONTAL_SWIPE_COMMIT_PX,
  axisRatio = HORIZONTAL_SWIPE_AXIS_RATIO,
) {
  const absX = Math.abs(Number(dx) || 0);
  const absY = Math.abs(Number(dy) || 0);
  if (absX < commitPx) return false;
  return absX > absY * axisRatio;
}

export function horizontalSwipeDirection(dx: number) {
  const offset = Number(dx) || 0;
  if (offset === 0) return 0;
  return offset < 0 ? 1 : -1;
}

export function startSwipeHoldTimer(
  state: HorizontalSwipeState | null,
  onHold?: ((nextState: HorizontalSwipeState) => void) | null,
  holdMs = HORIZONTAL_SWIPE_HOLD_MS,
) {
  if (!state) return () => {};
  const timer = window.setTimeout(() => {
    if (!state) return;
    state.held = true;
    if (typeof onHold === 'function') {
      onHold(state);
    }
  }, holdMs);
  return () => {
    window.clearTimeout(timer);
  };
}
