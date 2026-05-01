package store

import "testing"

func TestItemDedupCandidateListsBindingsContainersDatesAndReasoning(t *testing.T) {
	s := newTestStore(t)
	first, second := seedDedupActionItems(t, s)

	group, err := s.CreateItemDedupCandidate(ItemDedupCandidateOptions{
		Kind:       ItemKindAction,
		Score:      0.82,
		Confidence: 0.91,
		Outcome:    "Review budget",
		Reasoning:  "local model matched outcome text and source context",
		Detector:   "local-llm",
		Items: []ItemDedupCandidateItemInput{
			{ItemID: first.ID, Outcome: "Review the revised budget"},
			{ItemID: second.ID, Outcome: "Check the budget revision"},
		},
	})
	if err != nil {
		t.Fatalf("CreateItemDedupCandidate() error: %v", err)
	}

	groups, err := s.ListItemDedupCandidatesFiltered(ItemKindAction, ItemListFilter{})
	if err != nil {
		t.Fatalf("ListItemDedupCandidatesFiltered() error: %v", err)
	}
	if len(groups) != 1 || groups[0].ID != group.ID {
		t.Fatalf("groups = %#v, want candidate %d", groups, group.ID)
	}
	got := groups[0]
	if got.Reasoning != "local model matched outcome text and source context" || got.Detector != "local-llm" {
		t.Fatalf("reasoning/detector = %#v", got)
	}
	if len(got.Items) != 2 {
		t.Fatalf("members = %d, want 2", len(got.Items))
	}
	for _, member := range got.Items {
		if len(member.SourceBindings) != 1 {
			t.Fatalf("member %d bindings = %#v, want one binding", member.Item.ID, member.SourceBindings)
		}
		if len(member.SourceContainers) != 1 {
			t.Fatalf("member %d containers = %#v, want one container", member.Item.ID, member.SourceContainers)
		}
		if len(member.Dates) == 0 {
			t.Fatalf("member %d dates empty", member.Item.ID)
		}
	}
}

func TestItemDedupKeepSeparateRecordsNonDuplicateDecision(t *testing.T) {
	s := newTestStore(t)
	first, second := seedDedupActionItems(t, s)
	group := mustDedupCandidate(t, s, ItemKindAction, first.ID, second.ID)

	updated, err := s.ApplyItemDedupDecision(group.ID, ItemDedupActionKeepSeparate, nil)
	if err != nil {
		t.Fatalf("ApplyItemDedupDecision(keep_separate) error: %v", err)
	}
	if updated.State != ItemDedupStateKeepSeparate || updated.ReviewedAt == nil {
		t.Fatalf("updated group = %#v, want keep_separate with reviewed_at", updated)
	}
	open, err := s.ListItemDedupCandidatesFiltered("", ItemListFilter{})
	if err != nil {
		t.Fatalf("ListItemDedupCandidatesFiltered() error: %v", err)
	}
	if len(open) != 0 {
		t.Fatalf("open candidates = %#v, want none", open)
	}
}

func TestItemDedupMergePreservesBindingsOnCanonicalItem(t *testing.T) {
	s := newTestStore(t)
	first, second := seedDedupActionItems(t, s)
	group := mustDedupCandidate(t, s, ItemKindAction, first.ID, second.ID)

	updated, err := s.ApplyItemDedupDecision(group.ID, ItemDedupActionMerge, &first.ID)
	if err != nil {
		t.Fatalf("ApplyItemDedupDecision(merge) error: %v", err)
	}
	if updated.State != ItemDedupStateMerged || updated.CanonicalItemID == nil || *updated.CanonicalItemID != first.ID {
		t.Fatalf("updated group = %#v, want merged into %d", updated, first.ID)
	}
	bindings, err := s.GetBindingsByItem(first.ID)
	if err != nil {
		t.Fatalf("GetBindingsByItem(canonical) error: %v", err)
	}
	if len(bindings) != 2 {
		t.Fatalf("canonical bindings = %#v, want both source bindings", bindings)
	}
	loserBindings, err := s.GetBindingsByItem(second.ID)
	if err != nil {
		t.Fatalf("GetBindingsByItem(loser) error: %v", err)
	}
	if len(loserBindings) != 0 {
		t.Fatalf("loser bindings = %#v, want none", loserBindings)
	}
}

func TestItemDedupReviewLaterStaysVisibleInTailQueue(t *testing.T) {
	s := newTestStore(t)
	first, second := seedDedupActionItems(t, s)
	third, fourth := seedDedupProjectItems(t, s)
	later := mustDedupCandidate(t, s, ItemKindAction, first.ID, second.ID)
	open := mustDedupCandidate(t, s, ItemKindProject, third.ID, fourth.ID)
	if _, err := s.ApplyItemDedupDecision(later.ID, ItemDedupActionReviewLater, nil); err != nil {
		t.Fatalf("ApplyItemDedupDecision(review_later) error: %v", err)
	}

	groups, err := s.ListItemDedupCandidatesFiltered("", ItemListFilter{})
	if err != nil {
		t.Fatalf("ListItemDedupCandidatesFiltered() error: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("groups len = %d, want 2", len(groups))
	}
	if groups[0].ID != open.ID || groups[1].ID != later.ID {
		t.Fatalf("queue order = [%d %d], want open %d before later %d", groups[0].ID, groups[1].ID, open.ID, later.ID)
	}
	if groups[1].State != ItemDedupStateReviewLater {
		t.Fatalf("tail state = %q, want review_later", groups[1].State)
	}
}

func TestItemDedupSeparatesActionAndProjectCandidates(t *testing.T) {
	s := newTestStore(t)
	actionA, actionB := seedDedupActionItems(t, s)
	projectA, projectB := seedDedupProjectItems(t, s)
	actionGroup := mustDedupCandidate(t, s, ItemKindAction, actionA.ID, actionB.ID)
	projectGroup := mustDedupCandidate(t, s, ItemKindProject, projectA.ID, projectB.ID)

	actions, err := s.ListItemDedupCandidatesFiltered(ItemKindAction, ItemListFilter{})
	if err != nil {
		t.Fatalf("ListItemDedupCandidatesFiltered(action) error: %v", err)
	}
	if len(actions) != 1 || actions[0].ID != actionGroup.ID {
		t.Fatalf("action groups = %#v, want %d", actions, actionGroup.ID)
	}
	projects, err := s.ListItemDedupCandidatesFiltered(ItemKindProject, ItemListFilter{})
	if err != nil {
		t.Fatalf("ListItemDedupCandidatesFiltered(project) error: %v", err)
	}
	if len(projects) != 1 || projects[0].ID != projectGroup.ID {
		t.Fatalf("project groups = %#v, want %d", projects, projectGroup.ID)
	}
	if _, err := s.CreateItemDedupCandidate(ItemDedupCandidateOptions{
		Kind:  ItemKindAction,
		Items: []ItemDedupCandidateItemInput{{ItemID: actionA.ID}, {ItemID: projectA.ID}},
	}); err == nil {
		t.Fatal("mixed action/project candidate error = nil, want rejection")
	}
}

func mustDedupCandidate(t *testing.T, s *Store, kind string, firstID, secondID int64) ItemDedupCandidateGroup {
	t.Helper()
	group, err := s.CreateItemDedupCandidate(ItemDedupCandidateOptions{
		Kind:       kind,
		Confidence: 0.9,
		Items:      []ItemDedupCandidateItemInput{{ItemID: firstID}, {ItemID: secondID}},
	})
	if err != nil {
		t.Fatalf("CreateItemDedupCandidate(%s) error: %v", kind, err)
	}
	return group
}

func seedDedupActionItems(t *testing.T, s *Store) (Item, Item) {
	t.Helper()
	first := mustDedupItem(t, s, "Review budget", ItemKindAction, "todoist", "task-1", "Finance")
	second := mustDedupItem(t, s, "Check budget revision", ItemKindAction, ExternalProviderGmail, "msg-7", "Planning")
	return first, second
}

func seedDedupProjectItems(t *testing.T, s *Store) (Item, Item) {
	t.Helper()
	first := mustDedupItem(t, s, "Ship review queue", ItemKindProject, ExternalProviderBear, "brain/one.md", "Brain")
	second := mustDedupItem(t, s, "Review queue outcome", ItemKindProject, ExternalProviderTodoist, "project-2", "Work")
	return first, second
}

func mustDedupItem(t *testing.T, s *Store, title, kind, provider, remoteID, container string) Item {
	t.Helper()
	item, err := s.CreateItem(title, ItemOptions{Kind: kind, State: ItemStateNext})
	if err != nil {
		t.Fatalf("CreateItem(%q) error: %v", title, err)
	}
	account, err := s.CreateExternalAccount(SphereWork, provider, provider+" "+remoteID, map[string]any{})
	if err != nil {
		t.Fatalf("CreateExternalAccount(%s) error: %v", provider, err)
	}
	remoteAt := "2026-04-30T09:15:00Z"
	if _, err := s.UpsertExternalBinding(ExternalBinding{
		AccountID:       account.ID,
		Provider:        account.Provider,
		ObjectType:      "task",
		RemoteID:        remoteID,
		ItemID:          &item.ID,
		ContainerRef:    &container,
		RemoteUpdatedAt: &remoteAt,
	}); err != nil {
		t.Fatalf("UpsertExternalBinding(%s) error: %v", remoteID, err)
	}
	return item
}
