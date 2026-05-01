// Slopshell Playwright harness — GTD dedup review routes.
function handleDedupHarnessRoute(u, opts) {
  if (/\/api\/items\/dedup\/\d+\/[^/?]+(?:\?|$)/.test(u) && opts?.method === 'POST') {
    return handleDedupActionRoute(u, opts);
  }
  if (/\/api\/items\/dedup(?:\?|$)/.test(u)) {
    return handleDedupListRoute(u, opts);
  }
  return null;
}

function handleDedupActionRoute(u, opts) {
  const match = u.match(/\/api\/items\/dedup\/(\d+)\/([^/?]+)(?:\?|$)/);
  const candidateID = Number(match?.[1] || 0);
  const action = decodeURIComponent(String(match?.[2] || ''));
  let body = {};
  try { body = JSON.parse(String(opts?.body || '{}')); } catch (_) { body = {}; }
  const candidates = Array.isArray(window.__itemDedupCandidates) ? window.__itemDedupCandidates : [];
  const index = candidates.findIndex((entry) => Number(entry?.id || 0) === candidateID);
  const group = index >= 0 ? candidates[index] : null;
  window.__harnessLog.push({
    type: 'api_fetch',
    action: 'dedup_action',
    method: opts?.method || 'POST',
    url: u,
    payload: { candidate_id: candidateID, action, ...body },
  });
  if (!group) return new Response('dedup candidate not found', { status: 404 });
  if (action === 'review_later') {
    group.state = 'review_later';
  } else {
    group.state = action === 'merge' ? 'merged' : 'keep_separate';
    candidates.splice(index, 1);
  }
  return new Response(JSON.stringify({ ok: true, group, action }), { status: 200 });
}

function handleDedupListRoute(u, opts) {
  const candidates = Array.isArray(window.__itemDedupCandidates) ? window.__itemDedupCandidates : [];
  const sphere = requestedSphere(u);
  const filters = requestedItemFilters(u);
  const groups = candidates
    .filter((group) => ['open', 'review_later'].includes(String(group?.state || '').trim().toLowerCase()))
    .filter((group) => dedupGroupMatchesFilters(group, sphere, filters))
    .sort((a, b) => dedupGroupSortKey(a).localeCompare(dedupGroupSortKey(b)));
  window.__harnessLog.push({
    type: 'api_fetch',
    action: 'dedup_review',
    method: opts?.method || 'GET',
    url: u,
    payload: { count: groups.length },
  });
  return new Response(JSON.stringify({ ok: true, groups, total: groups.length }), { status: 200 });
}

function dedupGroupMatchesFilters(group, sphere, filters) {
  const items = Array.isArray(group?.items) ? group.items : [];
  return items.some((member) => matchesItemFilters(member?.item || {}, sphere, filters));
}

function dedupGroupSortKey(group) {
  const stateRank = String(group?.state || '').trim().toLowerCase() === 'review_later' ? '1' : '0';
  return `${stateRank}:${String(group?.detected_at || '')}:${String(group?.id || '')}`;
}
