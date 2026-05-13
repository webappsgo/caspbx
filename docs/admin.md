# Admin Panel

## Current Status

The admin foundation is live for authenticated server-admin sessions and mirrored admin API tokens.

## Implemented Admin Surface

- `/admin` for server-admin login and profile access
- `/admin/server/settings`, `/admin/server/users`, and `/admin/server/security/auth` as protected admin foundations
- `/admin/server/domains` for custom-domain moderation and lifecycle control
- `/admin/server/asterisk` for capability-aware Asterisk administration
- `/admin/server/asterisk/extensions`, `/trunks`, `/routes`, `/queues`, `/conferences`, `/ivrs`, `/prompt-assignments`, `/provisioning-profiles`, and `/apply-preview` for persisted PBX object management
- `/admin/server/asterisk/operator`, `/operator/trunks`, `/operator/conferences`, and `/operator/parked-calls` for operator runtime visibility
- `/admin/server/asterisk/callcenter`, `/callcenter/queues`, `/callcenter/agents`, `/callcenter/supervisor-actions`, and `/callcenter/supervisor-actions/preview` for queue and supervisor operations
- the authenticated user portal now lives alongside the admin surfaces under `/users` and mirrors to `/api/v1/users` for communications dashboard, contacts, call history, voicemail, messages, presence, webphone, and settings

## Implemented Asterisk Admin Surfaces

- `overview`
- `health`
- `capabilities`
- `modules`
- `apply`
- capability-gated media/fax/messaging/conference/queue/hardware/browser surfaces when enabled by the active capability model
- persisted PBX entity collections and detail routes for extensions, trunks, routes, queues, conferences, IVRs, prompt assignments, provisioning profiles, and apply-preview validation
- operator and call-center surfaces for dashboard, trunk/conference/parking visibility, queue/agent wallboards, supervisor action catalogs, and action previews

## Admin Rules Reflected in Code

- role-aware access and secure defaults
- capability-driven UI visibility
- isolated server-admin session handling
- no routine terminal dependence after installation

## Product Direction

The goal is a single coherent admin experience that replaces fragmented PBX, fax, operator, reporting, and backend administration surfaces around Asterisk.
