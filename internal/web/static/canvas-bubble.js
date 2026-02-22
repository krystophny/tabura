import { marked } from './vendor/marked.esm.js';
import { escapeHtml, sanitizeHtml, clearLineHighlight } from './canvas.js';

function bubbleParse(markdownText) {
  return marked.parse(markdownText || '');
}

let activeBubble = null;
let activeThreadKey = '';
let bubbleSendFn = null;
let bubbleVoiceFn = null;

export function isAnnotationBubbleOpen() {
  return activeBubble !== null;
}

export function getActiveThreadKey() {
  return activeThreadKey;
}

export function openAnnotationBubble({ location, clientX, clientY, voiceAutoStart }) {
  closeAnnotationBubble();
  const isMobile = window.innerWidth < 600;
  const isWide = !isMobile && window.innerWidth >= 900;
  activeThreadKey = `ann-${Date.now()}`;

  const bubble = document.createElement('div');
  bubble.className = 'annotation-bubble';
  bubble.dataset.threadKey = activeThreadKey;

  const header = document.createElement('div');
  header.className = 'annotation-bubble-header';
  const locSpan = document.createElement('span');
  locSpan.className = 'annotation-bubble-location';
  locSpan.textContent = location
    ? `Line ${location.line} of "${location.title}"`
    : 'Annotation';
  const dismiss = document.createElement('button');
  dismiss.className = 'annotation-bubble-dismiss';
  dismiss.type = 'button';
  dismiss.setAttribute('aria-label', 'close');
  dismiss.textContent = '\u00d7';
  dismiss.addEventListener('click', (ev) => {
    ev.stopPropagation();
    closeAnnotationBubble();
  });
  header.appendChild(locSpan);
  header.appendChild(dismiss);

  const messages = document.createElement('div');
  messages.className = 'annotation-bubble-messages';

  const bar = document.createElement('div');
  bar.className = 'annotation-bubble-bar';
  const input = document.createElement('textarea');
  input.className = 'annotation-bubble-input';
  input.placeholder = 'Comment...';
  input.rows = 1;
  const send = document.createElement('button');
  send.className = 'annotation-bubble-send';
  send.type = 'button';
  send.setAttribute('aria-label', 'send');
  send.textContent = '\u25b6';
  bar.appendChild(input);
  bar.appendChild(send);

  bubble.appendChild(header);
  bubble.appendChild(messages);
  bubble.appendChild(bar);

  const submitBubble = () => {
    const text = input.value.trim();
    if (!text) return;
    appendBubbleMessage('user', text);
    input.value = '';
    input.style.height = 'auto';
    if (bubbleSendFn) {
      bubbleSendFn(text, activeThreadKey);
    }
  };

  send.addEventListener('click', (ev) => {
    ev.preventDefault();
    submitBubble();
  });

  input.addEventListener('keydown', (ev) => {
    if (ev.key === 'Enter' && !ev.shiftKey) {
      ev.preventDefault();
      ev.stopPropagation();
      submitBubble();
    }
  });

  input.addEventListener('input', () => {
    input.style.height = 'auto';
    input.style.height = `${Math.min(input.scrollHeight, 120)}px`;
  });

  // Voice hold on send button
  const HOLD_MS = 300;
  let sendHoldTimer = null;
  let sendHoldActive = false;
  let sendHoldIsTouch = false;
  const sendStartHold = (ev, isTouch) => {
    if (isTouch) ev.preventDefault();
    sendHoldActive = false;
    sendHoldIsTouch = isTouch;
    sendHoldTimer = window.setTimeout(() => {
      sendHoldTimer = null;
      sendHoldActive = true;
      if (bubbleVoiceFn) bubbleVoiceFn(true);
    }, HOLD_MS);
  };
  const sendEndHold = () => {
    if (sendHoldTimer) {
      clearTimeout(sendHoldTimer);
      sendHoldTimer = null;
      sendHoldIsTouch = false;
      return;
    }
    if (sendHoldActive) {
      sendHoldActive = false;
      sendHoldIsTouch = false;
      if (bubbleVoiceFn) bubbleVoiceFn(false);
    }
  };
  send.addEventListener('touchstart', (ev) => sendStartHold(ev, true), { passive: false });
  send.addEventListener('mousedown', (ev) => {
    if (ev.button !== 0 || sendHoldIsTouch) return;
    sendStartHold(ev, false);
  });
  window.addEventListener('touchend', () => { if (sendHoldIsTouch) sendEndHold(); }, { passive: true });
  window.addEventListener('touchcancel', () => { if (sendHoldIsTouch) sendEndHold(); });
  window.addEventListener('mouseup', () => { if (!sendHoldIsTouch) sendEndHold(); });

  // Voice hold on input textarea (PTT when empty)
  let inputHoldTimer = null;
  let inputHoldActive = false;
  let inputHoldIsTouch = false;
  const inputStartHold = (ev, isTouch) => {
    if (input.value.trim()) return;
    inputHoldActive = false;
    inputHoldIsTouch = isTouch;
    inputHoldTimer = window.setTimeout(() => {
      inputHoldTimer = null;
      inputHoldActive = true;
      if (isTouch) input.blur();
      input.classList.add('is-recording');
      if (bubbleVoiceFn) bubbleVoiceFn(true);
    }, HOLD_MS);
  };
  const inputEndHold = () => {
    if (inputHoldTimer) {
      clearTimeout(inputHoldTimer);
      inputHoldTimer = null;
      inputHoldIsTouch = false;
      return;
    }
    if (inputHoldActive) {
      inputHoldActive = false;
      inputHoldIsTouch = false;
      input.classList.remove('is-recording');
      if (bubbleVoiceFn) bubbleVoiceFn(false);
    }
  };
  input.addEventListener('touchstart', (ev) => inputStartHold(ev, true), { passive: true });
  input.addEventListener('mousedown', (ev) => {
    if (ev.button !== 0 || inputHoldIsTouch) return;
    inputStartHold(ev, false);
  });
  window.addEventListener('touchend', () => { if (inputHoldIsTouch) inputEndHold(); }, { passive: true });
  window.addEventListener('touchcancel', () => { if (inputHoldIsTouch) inputEndHold(); });
  window.addEventListener('mouseup', () => { if (!inputHoldIsTouch) inputEndHold(); });

  if (isMobile) {
    document.body.appendChild(bubble);
  } else if (isWide) {
    bubble.classList.add('annotation-side-panel');
    const viewport = document.getElementById('canvas-viewport');
    if (viewport) {
      viewport.classList.add('has-annotation-panel');
      viewport.appendChild(bubble);
    } else {
      document.body.appendChild(bubble);
    }
  } else {
    const canvasText = document.getElementById('canvas-text');
    const isCanvasVisible = canvasText && canvasText.offsetParent !== null;
    if (canvasText && isCanvasVisible) {
      if (window.getComputedStyle(canvasText).position === 'static') {
        canvasText.style.position = 'relative';
      }
      const rootRect = canvasText.getBoundingClientRect();
      let top = clientY - rootRect.top + canvasText.scrollTop + 8;
      let left = clientX - rootRect.left + canvasText.scrollLeft;
      const maxWidth = 340;
      if (left + maxWidth > rootRect.width) {
        left = Math.max(10, rootRect.width - maxWidth - 10);
      }
      if (left < 10) left = 10;
      bubble.style.top = `${top}px`;
      bubble.style.left = `${left}px`;
      canvasText.appendChild(bubble);
    } else {
      bubble.style.position = 'fixed';
      bubble.style.top = `${Math.min(clientY + 8, window.innerHeight - 300)}px`;
      bubble.style.left = `${Math.min(clientX, window.innerWidth - 350)}px`;
      document.body.appendChild(bubble);
    }
  }

  activeBubble = bubble;

  // Dismiss on click outside
  const outsideHandler = (ev) => {
    if (!activeBubble) return;
    if (activeBubble.contains(ev.target)) return;
    if (isWide) {
      const ct = document.getElementById('canvas-text');
      if (ct && ct.contains(ev.target)) return;
    }
    closeAnnotationBubble();
  };
  window.setTimeout(() => {
    document.addEventListener('click', outsideHandler, { capture: true });
    bubble._outsideHandler = outsideHandler;
  }, 50);

  // Dismiss on Escape
  const escHandler = (ev) => {
    if (ev.key === 'Escape') {
      ev.preventDefault();
      ev.stopPropagation();
      closeAnnotationBubble();
    }
  };
  document.addEventListener('keydown', escHandler, { capture: true });
  bubble._escHandler = escHandler;

  // Mobile swipe-down dismiss
  if (isMobile) {
    let touchStartY = 0;
    bubble.addEventListener('touchstart', (ev) => {
      if (ev.touches.length === 1) {
        touchStartY = ev.touches[0].clientY;
      }
    }, { passive: true });
    bubble.addEventListener('touchend', (ev) => {
      if (ev.changedTouches.length === 1) {
        const dy = ev.changedTouches[0].clientY - touchStartY;
        if (dy > 60) closeAnnotationBubble();
      }
    }, { passive: true });
  }

  window.setTimeout(() => {
    input.focus();
  }, 50);

  if (voiceAutoStart && bubbleVoiceFn) {
    window.setTimeout(() => bubbleVoiceFn(true), 100);
  }
}

export function closeAnnotationBubble() {
  if (!activeBubble) return;
  const bubble = activeBubble;
  activeBubble = null;
  activeThreadKey = '';
  if (bubble._outsideHandler) {
    document.removeEventListener('click', bubble._outsideHandler, { capture: true });
  }
  if (bubble._escHandler) {
    document.removeEventListener('keydown', bubble._escHandler, { capture: true });
  }
  const viewport = document.getElementById('canvas-viewport');
  if (viewport) viewport.classList.remove('has-annotation-panel');
  bubble.remove();
  clearLineHighlight();
}

function appendBubbleMessage(role, text) {
  if (!activeBubble) return;
  const messages = activeBubble.querySelector('.annotation-bubble-messages');
  if (!messages) return;
  const row = document.createElement('div');
  row.className = `annotation-bubble-msg annotation-bubble-msg-${role}`;
  if (role === 'assistant') {
    row.innerHTML = sanitizeHtml(bubbleParse(text));
  } else {
    row.textContent = text;
  }
  messages.appendChild(row);
  messages.scrollTop = messages.scrollHeight;
  return row;
}

let pendingAssistantRow = null;
let pendingAssistantText = '';

export function routeBubbleEvent(payload) {
  if (!activeBubble) return;
  const type = String(payload?.type || '').trim();

  if (type === 'turn_started') {
    pendingAssistantText = '';
    pendingAssistantRow = appendBubbleMessage('assistant', '_Thinking..._');
    return;
  }

  if (type === 'assistant_message') {
    pendingAssistantText = String(payload.message || '');
    if (pendingAssistantRow) {
      pendingAssistantRow.innerHTML = sanitizeHtml(bubbleParse(pendingAssistantText));
      const messages = activeBubble.querySelector('.annotation-bubble-messages');
      if (messages) messages.scrollTop = messages.scrollHeight;
    }
    return;
  }

  if (type === 'message_persisted' && String(payload.role || '') === 'assistant') {
    const text = String(payload.message || pendingAssistantText || '');
    if (pendingAssistantRow) {
      pendingAssistantRow.innerHTML = sanitizeHtml(bubbleParse(text));
      pendingAssistantRow.classList.remove('is-pending');
      const messages = activeBubble.querySelector('.annotation-bubble-messages');
      if (messages) messages.scrollTop = messages.scrollHeight;
    } else {
      appendBubbleMessage('assistant', text);
    }
    pendingAssistantRow = null;
    pendingAssistantText = '';
    return;
  }

  if (type === 'error') {
    const errText = String(payload.error || 'error');
    if (pendingAssistantRow) {
      pendingAssistantRow.textContent = errText;
      pendingAssistantRow.classList.add('annotation-bubble-msg-error');
    } else {
      const row = appendBubbleMessage('assistant', errText);
      if (row) row.classList.add('annotation-bubble-msg-error');
    }
    pendingAssistantRow = null;
    pendingAssistantText = '';
  }
}

export function setBubbleSendFn(fn) {
  bubbleSendFn = fn;
}

export function setBubbleVoiceFn(fn) {
  bubbleVoiceFn = fn;
}

export function appendBubbleTranscript(text) {
  if (!activeBubble) return;
  const input = activeBubble.querySelector('.annotation-bubble-input');
  if (!input) return;
  const needsSpace = input.value.trim() && !/[ \n]$/.test(input.value);
  input.value = `${input.value}${needsSpace ? ' ' : ''}${text}`;
  input.dispatchEvent(new Event('input', { bubbles: true }));
}
