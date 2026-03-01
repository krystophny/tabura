package plugins

import "context"

type HookProvider interface {
	Apply(ctx context.Context, req HookRequest) HookResult
	DecideMeetingPartner(ctx context.Context, req HookRequest) (MeetingPartnerDecision, bool)
}
