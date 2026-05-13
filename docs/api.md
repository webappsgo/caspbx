# API Reference

## Current Status

The API foundation is live and versioned under `/api/v1/...`.

## Implemented API Categories

- health and status: `/health`, `/healthz`, `/version`
- auth: `/api/v1/auth`
- user scope: `/api/v1/users`
- organization scope: `/api/v1/orgs/{slug}`
- admin scope: `/api/v1/admin`
- Asterisk admin scope: `/api/v1/admin/server/asterisk`
- custom-domain user/org/admin management under the matching user, org, and admin scopes

## Implemented User API Surfaces

- `/api/v1/users`
- `/api/v1/users/dashboard`
- `/api/v1/users/contacts`
- `/api/v1/users/call-history`
- `/api/v1/users/voicemail`
- `/api/v1/users/messages`
- `/api/v1/users/presence`
- `/api/v1/users/webphone`
- `/api/v1/users/communications/settings`
- user communications surfaces stay hidden behind capability-aware availability checks when the active Asterisk/PBX state cannot support them

## Implemented Asterisk API Surfaces

- `/api/v1/admin/server/asterisk`
- `/api/v1/admin/server/asterisk/health`
- `/api/v1/admin/server/asterisk/capabilities`
- `/api/v1/admin/server/asterisk/modules`
- `/api/v1/admin/server/asterisk/apply`
- `/api/v1/admin/server/asterisk/extensions`
- `/api/v1/admin/server/asterisk/trunks`
- `/api/v1/admin/server/asterisk/routes`
- `/api/v1/admin/server/asterisk/queues`
- `/api/v1/admin/server/asterisk/conferences`
- `/api/v1/admin/server/asterisk/ivrs`
- `/api/v1/admin/server/asterisk/prompt-assignments`
- `/api/v1/admin/server/asterisk/provisioning-profiles`
- `/api/v1/admin/server/asterisk/apply-preview`
- `/api/v1/admin/server/asterisk/operator`
- `/api/v1/admin/server/asterisk/operator/trunks`
- `/api/v1/admin/server/asterisk/operator/conferences`
- `/api/v1/admin/server/asterisk/operator/parked-calls`
- `/api/v1/admin/server/asterisk/callcenter`
- `/api/v1/admin/server/asterisk/callcenter/queues`
- `/api/v1/admin/server/asterisk/callcenter/agents`
- `/api/v1/admin/server/asterisk/callcenter/supervisor-actions`
- `/api/v1/admin/server/asterisk/callcenter/supervisor-actions/preview`
- capability-gated media/fax/messaging/conference/queue/hardware/browser surfaces

## Documentation Policy

This page should be updated as new real endpoints land so the docs always match the codebase.
