// Slopshell Playwright harness — GTD drift action route.
function handleDriftHarnessRoute(u, opts) {
  if (!/\/api\/items\/drift\/\d+\/[^/?]+(?:\?|$)/.test(u) || opts?.method !== 'POST') {
    return null;
  }
  const match = u.match(/\/api\/items\/drift\/(\d+)\/([^/?]+)(?:\?|$)/);
  const driftID = Number(match?.[1] || 0);
  const action = decodeURIComponent(String(match?.[2] || ''));
  const itemData = window.__itemSidebarData || defaultItemSidebarData();
  const review = Array.isArray(itemData.review) ? itemData.review : [];
  const index = review.findIndex((item) => Number(item?.drift_id || 0) === driftID);
  const drift = index >= 0 ? review.splice(index, 1)[0] : null;
  window.__harnessLog.push({
    type: 'api_fetch',
    action: 'drift_action',
    method: opts?.method || 'POST',
    url: u,
    payload: { drift_id: driftID, action },
  });
  if (!drift) return new Response('drift not found', { status: 404 });
  if (action === 'take_upstream') {
    drift.state = String(drift.upstream_state || drift.state || 'review');
    drift.title = String(drift.upstream_title || drift.title || '');
  }
  return new Response(JSON.stringify({ ok: true, drift, action }), { status: 200 });
}
