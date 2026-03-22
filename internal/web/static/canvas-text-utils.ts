export function lineFromOffset(lines: string[], charOffset: number) {
  let charCount = 0;
  for (let i = 0; i < lines.length; i += 1) {
    if (charCount + lines[i].length >= charOffset) {
      return i + 1;
    }
    charCount += lines[i].length + 1;
  }
  return Math.max(1, lines.length);
}

export function compactAnchorText(raw: string, maxChars = 240) {
  const text = String(raw || '').trim();
  if (!text) return '';
  const collapsed = text.replace(/\s+/g, ' ').trim();
  if (collapsed.length <= maxChars) return collapsed;
  return `${collapsed.slice(0, maxChars)}...`;
}

export function surroundingTextForLine(lines: string[], line: number) {
  const lineNumber = Number.parseInt(String(line || ''), 10);
  if (!Array.isArray(lines) || lines.length === 0 || !Number.isFinite(lineNumber) || lineNumber <= 0) {
    return '';
  }
  const start = Math.max(0, lineNumber - 2);
  const end = Math.min(lines.length, lineNumber + 1);
  return lines
    .slice(start, end)
    .map((entry, index) => `${start + index + 1}: ${String(entry || '')}`)
    .join('\n')
    .trim();
}

function textRangeFromClientPoint(clientX: number, clientY: number) {
  if (typeof document.caretRangeFromPoint === 'function') {
    return document.caretRangeFromPoint(clientX, clientY);
  }
  if (typeof document.caretPositionFromPoint === 'function') {
    const caret = document.caretPositionFromPoint(clientX, clientY);
    if (!caret) return null;
    const range = document.createRange();
    range.setStart(caret.offsetNode, caret.offset);
    range.collapse(true);
    return range;
  }
  return null;
}

export function textRangeFromPointInRoot(root: HTMLElement | null, clientX: number, clientY: number) {
  const direct = textRangeFromClientPoint(clientX, clientY);
  if (!(root instanceof HTMLElement)) return direct;
  if (direct && root.contains(direct.startContainer)) return direct;

  const rect = root.getBoundingClientRect();
  if (clientX < rect.left || clientX > rect.right || clientY < rect.top || clientY > rect.bottom) {
    return direct;
  }

  const probeY = Math.max(rect.top + 1, Math.min(clientY, rect.bottom - 1));
  const probeXs = [
    Math.max(rect.left + 1, Math.min(clientX, rect.right - 1)),
    Math.max(rect.left + 1, Math.min(rect.left + 8, rect.right - 1)),
  ];
  for (const probeX of probeXs) {
    const probe = textRangeFromClientPoint(probeX, probeY);
    if (probe && root.contains(probe.startContainer)) {
      return probe;
    }
  }
  return direct;
}

export function estimateTextLineAtPoint(root: HTMLElement | null, clientY: number) {
  if (!(root instanceof HTMLElement)) return null;
  const range = document.createRange();
  range.selectNodeContents(root);
  const rects = Array.from(range.getClientRects()).filter((rect) => rect.width > 0 && rect.height > 0);
  if (rects.length === 0) return null;

  const lineRects = [];
  const topEpsilonPx = 1.5;
  for (const rect of rects) {
    const existing = lineRects.find((line) => Math.abs(line.top - rect.top) <= topEpsilonPx);
    if (existing) {
      existing.top = Math.min(existing.top, rect.top);
      existing.bottom = Math.max(existing.bottom, rect.bottom);
      existing.height = Math.max(existing.height, rect.height);
      continue;
    }
    lineRects.push({
      top: rect.top,
      bottom: rect.bottom,
      height: rect.height,
    });
  }
  lineRects.sort((a, b) => a.top - b.top);

  let nearestIndex = -1;
  let nearestDistance = Infinity;
  for (let i = 0; i < lineRects.length; i += 1) {
    const rect = lineRects[i];
    let distance = 0;
    if (clientY < rect.top) distance = rect.top - clientY;
    else if (clientY > rect.bottom) distance = clientY - rect.bottom;
    if (distance < nearestDistance) {
      nearestDistance = distance;
      nearestIndex = i;
    }
  }
  if (nearestIndex < 0) return null;
  return {
    line: nearestIndex + 1,
    top: lineRects[nearestIndex].top,
    height: lineRects[nearestIndex].height,
  };
}
