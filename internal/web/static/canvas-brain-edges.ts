import { apiURL } from './paths.js';
import type { BrainCanvasCard } from './canvas-brain-cards.js';

const EDGE_CONTROLS_CLASS = 'canvas-brain-edge-controls';
const EDGE_LAYER_CLASS = 'canvas-brain-edge-layer';
const SVG_NS = 'http://www.w3.org/2000/svg';

export interface BrainCanvasEdge {
  id: string;
  fromNode: string;
  toNode: string;
  label?: string;
  relation?: string;
  mode?: string;
}

interface BrainCanvasEdgeContext {
  workspaceID: string;
  reload: () => Promise<void>;
}

export function brainCanvasEdgeControls(section: HTMLElement): HTMLElement {
  const existing = section.querySelector(`.${EDGE_CONTROLS_CLASS}`);
  if (existing instanceof HTMLElement) return existing;
  const controls = document.createElement('div');
  controls.className = EDGE_CONTROLS_CLASS;
  const board = section.querySelector('.canvas-brain-cards-board');
  section.insertBefore(controls, board);
  return controls;
}

export function renderBrainCanvasEdges(board: HTMLElement, cards: BrainCanvasCard[], edges: BrainCanvasEdge[]) {
  const cardByID = new Map(cards.map((card) => [card.id, card]));
  const layer = document.createElementNS(SVG_NS, 'svg');
  layer.classList.add(EDGE_LAYER_CLASS);
  layer.setAttribute('aria-hidden', 'true');
  layer.appendChild(edgeMarker());
  for (const edge of edges) {
    const from = cardByID.get(edge.fromNode);
    const to = cardByID.get(edge.toNode);
    if (!from || !to) continue;
    renderEdgeLine(layer, edge, from, to);
  }
  board.appendChild(layer);
}

export function renderBrainCanvasEdgeControls(
  host: HTMLElement,
  cards: BrainCanvasCard[],
  edges: BrainCanvasEdge[],
  ctx: BrainCanvasEdgeContext,
) {
  host.replaceChildren();
  if (cards.length < 2) return;
  host.appendChild(edgeCreateForm(cards, ctx));
  if (edges.length) host.appendChild(edgeList(cards, edges, ctx));
}

function edgeMarker(): SVGDefsElement {
  const defs = document.createElementNS(SVG_NS, 'defs');
  const marker = document.createElementNS(SVG_NS, 'marker');
  marker.id = 'canvas-brain-edge-arrow';
  marker.setAttribute('viewBox', '0 0 10 10');
  marker.setAttribute('refX', '9');
  marker.setAttribute('refY', '5');
  marker.setAttribute('markerWidth', '7');
  marker.setAttribute('markerHeight', '7');
  marker.setAttribute('orient', 'auto-start-reverse');
  const path = document.createElementNS(SVG_NS, 'path');
  path.setAttribute('d', 'M 0 0 L 10 5 L 0 10 z');
  marker.appendChild(path);
  defs.appendChild(marker);
  return defs;
}

function renderEdgeLine(layer: SVGSVGElement, edge: BrainCanvasEdge, from: BrainCanvasCard, to: BrainCanvasCard) {
  const x1 = from.x + from.width / 2;
  const y1 = from.y + from.height / 2;
  const x2 = to.x + to.width / 2;
  const y2 = to.y + to.height / 2;
  const line = document.createElementNS(SVG_NS, 'line');
  line.classList.add('canvas-brain-edge');
  line.dataset.edgeId = edge.id;
  line.dataset.edgeMode = edge.mode || 'visual';
  line.setAttribute('x1', String(x1));
  line.setAttribute('y1', String(y1));
  line.setAttribute('x2', String(x2));
  line.setAttribute('y2', String(y2));
  line.setAttribute('marker-end', 'url(#canvas-brain-edge-arrow)');
  layer.appendChild(line);
  renderEdgeLabel(layer, edge, (x1 + x2) / 2, (y1 + y2) / 2);
}

function renderEdgeLabel(layer: SVGSVGElement, edge: BrainCanvasEdge, x: number, y: number) {
  const text = edge.label || edge.relation || '';
  if (!text) return;
  const label = document.createElementNS(SVG_NS, 'text');
  label.classList.add('canvas-brain-edge-label');
  label.dataset.edgeId = edge.id;
  label.setAttribute('x', String(x));
  label.setAttribute('y', String(y - 8));
  label.textContent = text;
  layer.appendChild(label);
}

function edgeCreateForm(cards: BrainCanvasCard[], ctx: BrainCanvasEdgeContext): HTMLFormElement {
  const form = document.createElement('form');
  form.className = 'canvas-brain-edge-form';
  const source = cardSelect(cards, 'From');
  const target = cardSelect(cards, 'To');
  const label = edgeInput('Label', 'canvas-brain-edge-label-input');
  const mode = edgeModeSelect();
  const relation = edgeInput('relation_slug', 'canvas-brain-edge-relation');
  const status = edgeStatus();
  form.append(source, target, label, mode, relation, submitButton('Add edge'), status);
  form.addEventListener('submit', (event) => {
    event.preventDefault();
    void createEdge(ctx, source.value, target.value, label.value, mode.value, relation.value, status);
  });
  return form;
}

function edgeList(cards: BrainCanvasCard[], edges: BrainCanvasEdge[], ctx: BrainCanvasEdgeContext): HTMLElement {
  const list = document.createElement('div');
  list.className = 'canvas-brain-edge-list';
  const cardByID = new Map(cards.map((card) => [card.id, card]));
  edges.forEach((edge) => list.appendChild(edgeRow(cardByID, edge, ctx)));
  return list;
}

function edgeRow(cardByID: Map<string, BrainCanvasCard>, edge: BrainCanvasEdge, ctx: BrainCanvasEdgeContext): HTMLElement {
  const row = document.createElement('div');
  row.className = 'canvas-brain-edge-row';
  row.dataset.edgeId = edge.id;
  const title = document.createElement('span');
  title.className = 'canvas-brain-edge-title';
  title.textContent = edgeTitle(cardByID, edge);
  const status = edgeStatus();
  row.append(title);
  if ((edge.mode || 'visual') === 'visual') row.append(...promotionControls(edge, ctx, status));
  row.append(deleteButton(edge, ctx, status), status);
  return row;
}

function promotionControls(edge: BrainCanvasEdge, ctx: BrainCanvasEdgeContext, status: HTMLElement): HTMLElement[] {
  const label = edgeInput('Label', 'canvas-brain-edge-label-input', edge.label || '');
  const relation = edgeInput('relation_slug', 'canvas-brain-edge-relation', edge.relation || '');
  const promote = actionButton('Promote');
  promote.addEventListener('click', () => {
    void promoteEdge(ctx, edge.id, label.value, relation.value, status);
  });
  return [label, relation, promote];
}

function cardSelect(cards: BrainCanvasCard[], label: string): HTMLSelectElement {
  const select = document.createElement('select');
  select.className = 'canvas-brain-edge-card';
  select.setAttribute('aria-label', label);
  cards.forEach((card) => {
    const option = document.createElement('option');
    option.value = card.id;
    option.textContent = card.title || card.id;
    select.appendChild(option);
  });
  return select;
}

function edgeModeSelect(): HTMLSelectElement {
  const select = document.createElement('select');
  select.className = 'canvas-brain-edge-mode';
  select.setAttribute('aria-label', 'Edge mode');
  [['visual', 'Visual'], ['semantic', 'Semantic']].forEach(([value, label]) => {
    const option = document.createElement('option');
    option.value = value;
    option.textContent = label;
    select.appendChild(option);
  });
  return select;
}

function edgeInput(placeholder: string, className: string, value = ''): HTMLInputElement {
  const input = document.createElement('input');
  input.className = className;
  input.placeholder = placeholder;
  input.value = value;
  input.autocomplete = 'off';
  return input;
}

function submitButton(text: string): HTMLButtonElement {
  const button = actionButton(text);
  button.type = 'submit';
  return button;
}

function actionButton(text: string): HTMLButtonElement {
  const button = document.createElement('button');
  button.type = 'button';
  button.className = 'canvas-brain-edge-action';
  button.textContent = text;
  return button;
}

function deleteButton(edge: BrainCanvasEdge, ctx: BrainCanvasEdgeContext, status: HTMLElement): HTMLButtonElement {
  const button = actionButton('Delete');
  button.classList.add('canvas-brain-edge-delete');
  button.addEventListener('click', () => {
    void deleteEdge(ctx, edge.id, status);
  });
  return button;
}

function edgeStatus(): HTMLElement {
  const status = document.createElement('span');
  status.className = 'canvas-brain-edge-status';
  status.setAttribute('role', 'status');
  return status;
}

function edgeTitle(cardByID: Map<string, BrainCanvasCard>, edge: BrainCanvasEdge): string {
  const from = cardByID.get(edge.fromNode)?.title || edge.fromNode;
  const to = cardByID.get(edge.toNode)?.title || edge.toNode;
  const label = edge.relation || edge.label || edge.mode || 'edge';
  return `${from} -> ${to} (${label})`;
}

async function createEdge(ctx: BrainCanvasEdgeContext, fromNode: string, toNode: string, label: string, mode: string, relation: string, status: HTMLElement) {
  await mutateEdge(ctx, 'edges', 'POST', { fromNode, toNode, label, mode, relation }, status);
}

async function promoteEdge(ctx: BrainCanvasEdgeContext, edgeID: string, label: string, relation: string, status: HTMLElement) {
  await mutateEdge(ctx, `edges/${encodeURIComponent(edgeID)}/promote`, 'POST', { label, relation }, status);
}

async function deleteEdge(ctx: BrainCanvasEdgeContext, edgeID: string, status: HTMLElement) {
  await mutateEdge(ctx, `edges/${encodeURIComponent(edgeID)}`, 'DELETE', undefined, status);
}

async function mutateEdge(ctx: BrainCanvasEdgeContext, path: string, method: string, body: Record<string, unknown> | undefined, status: HTMLElement) {
  status.textContent = '';
  const resp = await fetch(apiURL(`workspaces/${encodeURIComponent(ctx.workspaceID)}/brain-canvas/${path}`), {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!resp.ok) {
    status.textContent = (await resp.text()).trim() || `HTTP ${resp.status}`;
    return;
  }
  await ctx.reload();
}
