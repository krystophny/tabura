import * as context from './app-context.js';
import { workspaceDisplayName } from './app-workspace-status.js';

const { refs, state, SPHERE_OPTIONS, SIDEBAR_SOURCE_FILTERS } = context;

const setActiveSphere = (...args) => refs.setActiveSphere(...args);
const showTextInput = (...args) => refs.showTextInput(...args);
const activeProject = (...args) => refs.activeProject(...args);
const refreshWorkspaceBrowser = (...args) => refs.refreshWorkspaceBrowser(...args);
const renderPrReviewFileList = (...args) => refs.renderPrReviewFileList(...args);
const loadItemSidebarView = (...args) => refs.loadItemSidebarView(...args);
const openItemSidebarView = (...args) => refs.openItemSidebarView(...args);
const sidebarTabLabel = (...args) => refs.sidebarTabLabel(...args);
const suppressSyntheticClick = (...args) => refs.suppressSyntheticClick(...args);

const ITEM_SIDEBAR_VIEWS = context.ITEM_SIDEBAR_VIEWS;

const SIDEBAR_SECTION_DRILLDOWN: Record<string, { section: string; view: string | null }> = {
  'project-items': { section: 'project_items', view: null },
  people: { section: 'people', view: 'waiting' },
  drift: { section: 'drift', view: 'review' },
  dedup: { section: 'dedup', view: null },
  'recent-meetings': { section: 'recent_meetings', view: 'review' },
};

function applySidebarSourceFilter(sourceID) {
  const cleanSource = String(sourceID || '').trim().toLowerCase();
  const current = String(state.itemSidebarFilters?.source || '').trim().toLowerCase();
  const nextSource = current === cleanSource ? '' : cleanSource;
  const nextFilters = { ...state.itemSidebarFilters, source: nextSource };
  void loadItemSidebarView(state.itemSidebarView, nextFilters);
}

function applySidebarSectionDrilldown(sectionID) {
  const config = SIDEBAR_SECTION_DRILLDOWN[String(sectionID || '')] || null;
  if (!config) return;
  const currentSection = String(state.itemSidebarFilters?.section || '').trim().toLowerCase();
  const nextSection = currentSection === config.section ? '' : config.section;
  const nextFilters = { ...state.itemSidebarFilters, section: nextSection };
  const targetView = nextSection && config.view ? config.view : state.itemSidebarView;
  void loadItemSidebarView(targetView, nextFilters);
}

export function toggleSidebarSecondary() {
  state.itemSidebarSecondaryOpen = !Boolean(state.itemSidebarSecondaryOpen);
  refs.renderPrReviewFileList?.();
}

function bindSidebarTabActivation(button, onActivate) {
  let lastTouchAt = 0;
  let touchStart = null;
  button.addEventListener('touchstart', (ev) => {
    const touch = ev.touches && ev.touches[0];
    if (!touch) return;
    touchStart = { x: touch.clientX, y: touch.clientY };
  }, { passive: true });
  button.addEventListener('touchcancel', () => {
    touchStart = null;
  });
  button.addEventListener('touchend', (ev) => {
    const touch = ev.changedTouches && ev.changedTouches[0];
    const start = touchStart;
    touchStart = null;
    if (!touch || !start) return;
    if (Math.abs(touch.clientX - start.x) > 10 || Math.abs(touch.clientY - start.y) > 10) {
      return;
    }
    ev.preventDefault();
    ev.stopPropagation();
    lastTouchAt = Date.now();
    suppressSyntheticClick();
    onActivate();
  }, { passive: false });
  button.addEventListener('click', (ev) => {
    if (Date.now() - lastTouchAt < 700) {
      ev.preventDefault();
      return;
    }
    onActivate();
  });
}

export function renderSidebarPrimary(list) {
  const primary = document.createElement('div');
  primary.className = 'sidebar-primary';
  primary.id = 'sidebar-primary';

  const sphereRow = document.createElement('div');
  sphereRow.className = 'sidebar-sphere-row';
  sphereRow.id = 'sidebar-sphere-row';
  sphereRow.setAttribute('role', 'group');
  sphereRow.setAttribute('aria-label', 'Active sphere');
  const activeSphere = String(state.activeSphere || '').trim().toLowerCase();
  for (const opt of SPHERE_OPTIONS) {
    const btn = document.createElement('button');
    btn.type = 'button';
    btn.className = 'sidebar-sphere-btn';
    btn.dataset.sphere = opt.id;
    if (activeSphere === opt.id) {
      btn.classList.add('is-active');
      btn.setAttribute('aria-current', 'true');
    }
    btn.textContent = opt.label;
    btn.title = `Switch to ${opt.label.toLowerCase()} sphere`;
    btn.addEventListener('click', () => { void setActiveSphere(opt.id); });
    sphereRow.appendChild(btn);
  }
  primary.appendChild(sphereRow);

  const pinRow = document.createElement('div');
  pinRow.className = 'sidebar-workspace-pin';
  pinRow.id = 'sidebar-workspace-pin';
  const project = activeProject();
  const pinIcon = document.createElement('span');
  pinIcon.className = 'sidebar-workspace-pin-icon';
  pinIcon.setAttribute('aria-hidden', 'true');
  pinIcon.textContent = '◉';
  pinRow.appendChild(pinIcon);
  const pinBody = document.createElement('span');
  pinBody.className = 'sidebar-workspace-pin-body';
  const pinKicker = document.createElement('span');
  pinKicker.className = 'sidebar-workspace-pin-kicker';
  pinKicker.textContent = 'Workspace';
  pinBody.appendChild(pinKicker);
  const pinName = document.createElement('span');
  pinName.className = 'sidebar-workspace-pin-name';
  const projectName = project
    ? String(workspaceDisplayName(project) || project?.name || project?.id || 'Workspace').trim() || 'Workspace'
    : 'No workspace';
  pinName.textContent = projectName;
  pinName.title = project ? String(project?.root_path || projectName) : 'No workspace pinned';
  pinBody.appendChild(pinName);
  pinRow.appendChild(pinBody);
  primary.appendChild(pinRow);

  const actionsRow = document.createElement('div');
  actionsRow.className = 'sidebar-primary-actions';
  const captureBtn = document.createElement('button');
  captureBtn.type = 'button';
  captureBtn.className = 'sidebar-capture-btn edge-btn';
  captureBtn.id = 'sidebar-capture-trigger';
  captureBtn.textContent = 'Capture';
  captureBtn.title = 'Quick capture an item to the inbox';
  captureBtn.addEventListener('click', () => {
    const x = Math.max(16, Math.floor(window.innerWidth / 2) - 140);
    const y = Math.max(40, Math.floor(window.innerHeight * 0.18));
    showTextInput(x, y, null);
  });
  actionsRow.appendChild(captureBtn);
  primary.appendChild(actionsRow);

  list.appendChild(primary);
}

export function renderSidebarSecondary(list) {
  const secondary = document.createElement('div');
  secondary.className = 'sidebar-secondary';
  secondary.id = 'sidebar-secondary';
  if (state.itemSidebarSecondaryOpen) {
    secondary.classList.add('is-open');
  }

  const toggle = document.createElement('button');
  toggle.type = 'button';
  toggle.className = 'sidebar-secondary-toggle';
  toggle.id = 'sidebar-secondary-toggle';
  toggle.setAttribute('aria-expanded', state.itemSidebarSecondaryOpen ? 'true' : 'false');
  toggle.setAttribute('aria-controls', 'sidebar-secondary-body');
  const caret = document.createElement('span');
  caret.className = 'sidebar-secondary-caret';
  caret.setAttribute('aria-hidden', 'true');
  caret.textContent = state.itemSidebarSecondaryOpen ? '▾' : '▸';
  const toggleLabel = document.createElement('span');
  toggleLabel.className = 'sidebar-secondary-toggle-label';
  toggleLabel.textContent = 'Filters & sources';
  toggle.appendChild(caret);
  toggle.appendChild(toggleLabel);
  toggle.addEventListener('click', () => { toggleSidebarSecondary(); });
  secondary.appendChild(toggle);

  const body = document.createElement('div');
  body.className = 'sidebar-secondary-body';
  body.id = 'sidebar-secondary-body';
  body.hidden = !state.itemSidebarSecondaryOpen;

  const sectionCounts: Record<string, number> = state.itemSidebarSectionCounts || {};
  const projectItemsCount = Number(sectionCounts.project_items_open || 0);
  const peopleCount = Number(sectionCounts.people_open || 0);
  const driftCount = Number(sectionCounts.drift_review || 0);
  const dedupCount = Number(sectionCounts.dedup_review || 0);
  const recentMeetingsCount = Number(sectionCounts.recent_meetings || 0);
  const activeSection = String(state.itemSidebarFilters?.section || '').trim().toLowerCase();

  const sections = [
    {
      id: 'project-items',
      label: 'Project items',
      count: projectItemsCount,
      sectionFilter: 'project_items',
      title: 'Filter to open project items (Item kind=project). Project items stay surfaced as filters, not Workspaces.',
    },
    {
      id: 'people',
      label: 'People',
      count: peopleCount,
      sectionFilter: 'people',
      title: 'Filter to delegated/awaited items: distinct people the active queue owes work to or waits on.',
    },
    {
      id: 'drift',
      label: 'Drift',
      count: driftCount,
      sectionFilter: 'drift',
      title: 'Filter to review-state items with a review_target set: source drift review backlog.',
    },
    {
      id: 'dedup',
      label: 'Dedup',
      count: dedupCount,
      sectionFilter: 'dedup',
      title: 'Filter to open items whose (source, source_ref) collides with another row: duplicate review backlog.',
    },
    {
      id: 'recent-meetings',
      label: 'Recent meetings (7d)',
      count: recentMeetingsCount,
      sectionFilter: 'recent_meetings',
      title: 'Filter the queue down to items linked to meeting transcripts or summaries created in the last seven days.',
    },
  ];
  for (const section of sections) {
    const row = document.createElement('button');
    row.type = 'button';
    row.className = 'sidebar-secondary-row';
    row.dataset.sectionId = section.id;
    row.title = section.title;
    if (section.count <= 0) {
      row.classList.add('is-empty');
    }
    if (section.sectionFilter && section.sectionFilter === activeSection) {
      row.classList.add('is-active');
      row.setAttribute('aria-pressed', 'true');
    } else {
      row.setAttribute('aria-pressed', 'false');
    }
    const labelEl = document.createElement('span');
    labelEl.className = 'sidebar-secondary-row-label';
    labelEl.textContent = section.label;
    row.appendChild(labelEl);
    const badge = document.createElement('span');
    badge.className = 'sidebar-secondary-row-count';
    badge.textContent = section.count > 0 ? String(section.count) : '—';
    row.appendChild(badge);
    row.addEventListener('click', () => {
      applySidebarSectionDrilldown(section.id);
    });
    body.appendChild(row);
  }

  const sourcesGroup = document.createElement('div');
  sourcesGroup.className = 'sidebar-secondary-sources';
  sourcesGroup.id = 'sidebar-secondary-sources';
  const sourcesLabel = document.createElement('div');
  sourcesLabel.className = 'sidebar-secondary-sources-label';
  sourcesLabel.textContent = 'Sources';
  sourcesGroup.appendChild(sourcesLabel);
  const sourcesPills = document.createElement('div');
  sourcesPills.className = 'sidebar-secondary-sources-pills';
  const activeSource = String(state.itemSidebarFilters?.source || '').trim().toLowerCase();
  for (const source of SIDEBAR_SOURCE_FILTERS) {
    const pill = document.createElement('button');
    pill.type = 'button';
    pill.className = 'sidebar-source-pill';
    pill.dataset.sourceId = source.id;
    if (activeSource === source.id) {
      pill.classList.add('is-active');
      pill.setAttribute('aria-pressed', 'true');
    } else {
      pill.setAttribute('aria-pressed', 'false');
    }
    pill.textContent = source.label;
    pill.title = `Filter by ${source.label} source container`;
    pill.addEventListener('click', () => { applySidebarSourceFilter(source.id); });
    sourcesPills.appendChild(pill);
  }
  sourcesGroup.appendChild(sourcesPills);
  body.appendChild(sourcesGroup);

  secondary.appendChild(body);
  list.appendChild(secondary);
}

export function renderSidebarTabs(list) {
  const tabs = document.createElement('div');
  tabs.className = 'sidebar-tabs';
  ITEM_SIDEBAR_VIEWS.forEach((view) => {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = 'sidebar-tab';
    if (state.fileSidebarMode !== 'workspace' && state.itemSidebarView === view) {
      button.classList.add('is-active');
    }
    button.textContent = sidebarTabLabel(view);
    const count = Number(state.itemSidebarCounts?.[view] || 0);
    if (count > 0) {
      const badge = document.createElement('span');
      badge.className = 'sidebar-tab-count';
      badge.textContent = String(count);
      button.appendChild(badge);
    }
    bindSidebarTabActivation(button, () => {
      void openItemSidebarView(view);
    });
    tabs.appendChild(button);
  });
  const filesButton = document.createElement('button');
  filesButton.type = 'button';
  filesButton.className = 'sidebar-tab';
  if (state.fileSidebarMode === 'workspace') {
    filesButton.classList.add('is-active');
  }
  filesButton.textContent = 'Files';
  bindSidebarTabActivation(filesButton, () => {
    state.fileSidebarMode = 'workspace';
    renderPrReviewFileList();
    if (!state.workspaceBrowserLoading && state.workspaceBrowserEntries.length === 0 && !state.workspaceBrowserError) {
      void refreshWorkspaceBrowser(false);
    }
  });
  tabs.appendChild(filesButton);
  list.appendChild(tabs);
}
