import { normalizeCanvasPath } from './canvas-visual.js';

export const canvasPageState = {
  key: '',
  kind: '',
  pageIndex: 0,
  pageCount: 0,
  textPages: [] as HTMLElement[],
  textMeta: [] as Array<Record<string, any>>,
  activeTextPage: null as HTMLElement | null,
};

export function resetCanvasPageState(kind = '', key = '') {
  canvasPageState.kind = String(kind || '').trim();
  canvasPageState.key = String(key || '').trim();
  canvasPageState.pageIndex = 0;
  canvasPageState.pageCount = 0;
  canvasPageState.textPages = [];
  canvasPageState.textMeta = [];
  canvasPageState.activeTextPage = null;
}

export function canvasEventPageKey(event: Record<string, any> | null) {
  if (!event || typeof event !== 'object') return '';
  const path = normalizeCanvasPath(String(event.path || '').trim());
  const title = String(event.title || '').trim();
  const kind = String(event.kind || '').trim().toLowerCase();
  return `${kind}:${path || title}`;
}

export function summarizePageText(raw: string, maxChars = 320) {
  const collapsed = String(raw || '').replace(/\s+/g, ' ').trim();
  if (!collapsed) return '';
  if (collapsed.length <= maxChars) return collapsed;
  return `${collapsed.slice(0, maxChars)}...`;
}

function canvasViewportRect() {
  const viewport = document.getElementById('canvas-viewport');
  if (!(viewport instanceof HTMLElement)) return null;
  const rect = viewport.getBoundingClientRect();
  if (rect.width <= 0 || rect.height <= 0) return null;
  return rect;
}

export function visibleCanvasCenterPoint() {
  const rect = canvasViewportRect();
  if (!rect) {
    return {
      x: Math.floor(window.innerWidth / 2),
      y: Math.floor(window.innerHeight / 2),
    };
  }
  return {
    x: Math.floor(rect.left + (rect.width / 2)),
    y: Math.floor(rect.top + (rect.height / 2)),
  };
}

export function pageIndicatorLabel(pageNumber: number, pageCount: number, noun = 'Page') {
  const current = Math.max(1, Math.trunc(pageNumber || 1));
  const total = Math.max(current, Math.trunc(pageCount || current));
  return `${noun} ${current} / ${total}`;
}

export function buildCanvasPageIndicator(label: string) {
  const indicator = document.createElement('div');
  indicator.className = 'canvas-page-indicator';
  indicator.textContent = String(label || '').trim();
  return indicator;
}

function textSourceLinesForNode(node: Node | null) {
  if (!(node instanceof Element)) return [];
  const values = [];
  const own = Number.parseInt(String((node as HTMLElement).dataset?.sourceLine || '').trim(), 10);
  if (Number.isFinite(own) && own > 0) {
    values.push(own);
  }
  node.querySelectorAll('[data-source-line]').forEach((entry) => {
    if (!(entry instanceof HTMLElement)) return;
    const line = Number.parseInt(String(entry.dataset.sourceLine || '').trim(), 10);
    if (Number.isFinite(line) && line > 0) {
      values.push(line);
    }
  });
  return values.sort((left, right) => left - right);
}

function finalizeTextPage(page: HTMLElement, pageIndex: number, pageCount: number) {
  const body = page.querySelector('.canvas-text-page-body');
  const text = summarizePageText(body instanceof HTMLElement ? body.innerText : page.innerText);
  const lines = textSourceLinesForNode(page);
  const startLine = lines.length > 0 ? lines[0] : 0;
  const endLine = lines.length > 0 ? lines[lines.length - 1] : 0;
  const indicator = buildCanvasPageIndicator(pageIndicatorLabel(pageIndex + 1, pageCount, 'Page'));
  page.appendChild(indicator);
  return {
    page,
    meta: {
      id: `text-page-${pageIndex + 1}`,
      label: startLine > 0 && endLine >= startLine
        ? `Lines ${startLine}-${endLine}`
        : pageIndicatorLabel(pageIndex + 1, pageCount, 'Page'),
      text,
      startLine,
      endLine,
    },
  };
}

function createTextPageShell() {
  const page = document.createElement('section');
  page.className = 'canvas-text-page';
  const body = document.createElement('div');
  body.className = 'canvas-text-page-body';
  page.appendChild(body);
  return { page, body };
}

export function paginateTextRoot(root: HTMLElement | null) {
  if (!(root instanceof HTMLElement)) {
    return { pages: [], meta: [] };
  }
  const nodes = Array.from(root.childNodes);
  if (nodes.length === 0) {
    return { pages: [], meta: [] };
  }
  const pageHeight = Math.max(240, root.clientHeight || root.getBoundingClientRect().height || 240);
  const pages = [];
  let shell = createTextPageShell();
  root.replaceChildren(shell.page);
  for (const node of nodes) {
    shell.body.appendChild(node);
    const overflowed = shell.body.scrollHeight > pageHeight + 1;
    if (!overflowed) continue;
    const movingNode = shell.body.lastChild;
    if (!movingNode) continue;
    if (shell.body.childNodes.length === 1) continue;
    shell.body.removeChild(movingNode);
    pages.push(shell.page);
    shell = createTextPageShell();
    root.replaceChildren(shell.page);
    shell.body.appendChild(movingNode);
  }
  pages.push(shell.page);
  const finalizedPages = [];
  const meta = [];
  const pageCount = pages.length;
  for (let index = 0; index < pageCount; index += 1) {
    const result = finalizeTextPage(pages[index], index, pageCount);
    finalizedPages.push(result.page);
    meta.push(result.meta);
  }
  root.replaceChildren();
  return { pages: finalizedPages, meta };
}

export function renderTextPageAt(root: HTMLElement | null, pageIndex: number) {
  if (!(root instanceof HTMLElement)) return false;
  const total = canvasPageState.textPages.length;
  if (total === 0) {
    root.replaceChildren();
    canvasPageState.pageIndex = 0;
    canvasPageState.pageCount = 0;
    canvasPageState.activeTextPage = null;
    return false;
  }
  const nextIndex = Math.max(0, Math.min(total - 1, Math.trunc(pageIndex || 0)));
  const page = canvasPageState.textPages[nextIndex];
  if (!(page instanceof HTMLElement)) return false;
  root.replaceChildren(page);
  canvasPageState.pageIndex = nextIndex;
  canvasPageState.pageCount = total;
  canvasPageState.activeTextPage = page;
  return true;
}

export function currentTextPageUnit() {
  if (canvasPageState.kind !== 'text' || canvasPageState.pageCount <= 0) return null;
  const index = Math.max(0, Math.min(canvasPageState.pageCount - 1, canvasPageState.pageIndex));
  const meta = canvasPageState.textMeta[index] || {};
  return {
    id: String(meta.id || `text-page-${index + 1}`),
    label: String(meta.label || pageIndicatorLabel(index + 1, canvasPageState.pageCount, 'Page')),
    text: String(meta.text || '').trim(),
    start_line: Number(meta.startLine || 0),
    end_line: Number(meta.endLine || 0),
  };
}
