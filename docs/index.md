# CASPBX

`caspbx` is a unified Asterisk communications platform in active foundation development.

## Current State

The repository currently contains:

- a research-backed product definition
- a live Go runtime foundation for auth, users, orgs, domains, and admin scopes
- build, test, and documentation foundations
- a capability-driven Asterisk admin foundation under `/admin/server/asterisk`
- persisted PBX object and apply-preview foundations for extensions, trunks, routes, queues, conferences, IVRs, prompt assignments, and provisioning profiles
- a live user communications foundation for dashboard, contacts, call history, voicemail, messages, presence, webphone, and communications settings
- a live operator and call-center foundation for operator dashboards, queue and agent wallboards, parked-call visibility, conference visibility, and supervisor action previews
- a plan for a full replacement of multiple Asterisk-adjacent administration surfaces

## What CASPBX Is Intended to Become

- complete PBX administration
- end-user communications control
- operator and supervisor switchboard
- fax administration and user workflows
- call-center live operations and reporting
- backend and hosting administration

## Documentation

- [Installation](installation.md)
- [Configuration](configuration.md)
- [API Reference](api.md)
- [CLI Reference](cli.md)
- [Admin Panel](admin.md)
- [Development](development.md)
