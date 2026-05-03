## Project description

`caspbx` is a complete Asterisk-based communications platform and management frontend. The project is intended to provide a unified web application for deploying, managing, and operating a full PBX and telephony environment built around Asterisk and its supporting backend services, including PBX administration, user-facing communications tools, fax workflows, conferencing, call center functionality, and tenant-aware management.

The platform targets administrators, resellers, organizations, and hosted PBX operators who need a single application to manage an Asterisk stack instead of stitching together separate PBX, operator, fax, contact-center, and user-control tools. The goal is to deliver a free, full-featured frontend that covers the full communications management surface commonly split across multiple products while integrating with Asterisk and the surrounding telephony/fax infrastructure required by a deployment.

`caspbx` does not use a plugin or marketplace-style module system. The product is intended to ship with the full built-in platform surface, including the complete PBX administration and communications feature set, as part of one integrated application.

The product must support both single-tenant deployments and hosted multi-tenant deployments from day one. The intended top-level model is a provider/platform administration layer above tenant/customer environments, with users and extensions living inside those tenant boundaries and organizations supported inside tenants where needed.

## Product scope summary

`caspbx` is effectively the complete communications operating layer for an Asterisk deployment. It is not just a PBX configuration editor and not just an end-user portal. It is the single product expected to replace the broad collection of separate web interfaces, dashboards, helper apps, fax tools, operator consoles, and routine telephony administration surfaces that administrators otherwise assemble around Asterisk.

The intended scope is "Asterisk on steroids" as one built-in product:

- a complete PBX administration suite
- a complete end-user communications control panel
- a complete operator and supervisor console
- a complete call-center administration and operations surface
- a complete conferencing and collaboration administration surface
- a complete fax administration and user workflow surface
- a complete media, prompts, IVR, routing, and automation surface
- a complete tenant/provider hosting surface
- a complete backend stack administration surface for the telephony components required by the deployment

The first shippable release is not intended to be a reduced starter edition. The goal is the full named product surface from the beginning, with sane defaults and integrated workflows rather than a small subset that later grows through plugins or premium upgrades.

## Research-backed replacement surface inventory

The researched replacement stack resolves into a set of concrete product surfaces that `caspbx` must absorb into one coherent platform. Internally, the most important parity references are:

1. **PBX administration surface**
    - The broad admin surface typically spread across a very large PBX module catalog
    - Includes connectivity, call flow applications, schedules, audio assets, voicemail administration, security, backups, certificates, reporting, and system operations

2. **End-user communications portal**
    - Personal voicemail access
    - Personal call history and recordings access
    - Personal call handling and profile settings
    - Self-service authentication and account maintenance
    - User dashboard/widgets and day-to-day communications controls

3. **Operator and receptionist switchboard**
    - Real-time extension presence
    - Real-time queue visibility
    - Park/trunk/conference visibility
    - Fast live call handling actions such as pickup, blind transfer, attended transfer, voicemail transfer, hangup, park, and click-to-dial
    - Supervisor controls such as spy, whisper, barge, recording control, and queue agent state management

4. **Fax management surface**
    - Inbox, outbox, send, receive, browser viewing, download, forwarding, archive, search, contacts, line assignment, and fax routing workflows
    - Multi-line, per-user, and tenant-aware fax administration
    - Fax document management, categorization, keywords, and history visibility

5. **Call-center reporting and supervision surface**
    - Real-time queue load and wallboard visibility
    - Agent session and pause state visibility
    - Historical queue reports, SLA views, answered/unanswered/abandoned reporting, drill-down details, and recording access from reports

6. **System and operational administration surface**
    - Health dashboards, service state, security status, mail/fax/backend readiness, backup/restore, certificate workflows, firewall/intrusion posture, and other routine platform operations
    - The day-to-day operational functionality administrators often end up splitting across separate PBX admin, OS admin, security, and service-status interfaces

7. **Hosting and provider administration surface**
    - Single-tenant and hosted multi-tenant operation
    - Delegated customer administration
    - Organization layering where needed
    - Domain, branding, routing scope, and service boundary controls

## Research-backed roles and jobs-to-be-done

The replacement scope is not just a feature checklist. It must satisfy the real jobs different operators perform.

1. **Platform / System Administrator**
    - Provision extensions, endpoints, trunks, routes, schedules, queues, IVRs, conferences, recordings, voicemail, and security settings
    - Maintain certificates, mail delivery, backups, access controls, and backend service health
    - Bulk-manage users, devices, and telephony objects at provider or tenant scale

2. **Tenant / Customer Administrator**
    - Run one customer environment without needing host-level shell access
    - Manage tenant users, extensions, fax routing, queues, schedules, announcements, recordings, and communications settings

3. **Receptionist / Operator**
    - See live extension status
    - Handle high call volume quickly
    - Transfer, park, recover, and monitor calls from one fast surface
    - Search contacts and act in real time without leaving the switchboard

4. **Call Center Supervisor**
    - Watch queues live
    - Manage agent state during shifts
    - Coach agents with whisper/barge/spy-style interventions
    - Review historical performance, queue behavior, and recordings from one reporting surface

5. **End User / Extension User**
    - Access voicemail, call history, recordings, contacts, conferencing, webphone, and personal call handling controls
    - Manage personal profile and authentication workflows

6. **Fax User**
    - Send faxes, receive faxes, forward them by mail, archive them, search history, and manage fax contacts without using separate fax software

## Minimum parity reference by replacement surface

The minimum parity floor should be evaluated against concrete replacement surfaces rather than broad category labels.

1. **PBX administration parity floor**
    - Extensions and endpoint provisioning
    - Trunks and transport configuration
    - Inbound and outbound routing
    - IVRs, ring groups, queues, conferences, parking, announcements, directory, and time-based routing
    - Voicemail administration, music on hold, system recordings, TTS-backed prompt creation, feature code administration
    - User and group management, backup/restore, certificate handling, firewall/security controls, blacklist controls, bulk import/export, reporting, dashboards, and backend module/service administration

2. **End-user portal parity floor**
    - Login and self-service password reset
    - Voicemail inbox/playback/download/delete/greeting management
    - Personal call history with recording access
    - Personal profile, preferences, timezone/language, and communications settings
    - Personal dashboard and communications widgets

3. **Operator panel parity floor**
    - Live extension board with presence/state indicators
    - Queue, trunk, conference, and park visibility
    - Pickup, dial, hangup, blind transfer, attended transfer, voicemail transfer, and park operations
    - Supervisor-only actions for spy, whisper, barge, force recording, and queue-agent control
    - Fast phonebook/contact access inside the live panel

4. **Fax UI parity floor**
    - Fax inbox, outbox, send, browser view, download, email forward, archive, search, address book, line selection, and user/admin controls
    - Metadata, categorization, and auditability for fax workflows
    - Multi-line and role-aware fax management

5. **Call-center reporting parity floor**
    - Real-time wallboards
    - Historical SLA/distribution/answered/unanswered/abandoned/session/pause reporting
    - Recording access from report views
    - Exportable reports and supervisor-friendly drill-down workflows

6. **Operational environment parity floor**
    - Health/status visibility across communications services
    - Integrated mail/fax/backend operational awareness
    - Security posture visibility
    - Backup and recovery workflows
    - Enough platform administration to avoid routine dependence on separate day-to-day system-management products

## Project variables

project_name: caspbx
project_org: casapps
internal_name: caspbx
admin_path: admin
asterisk_admin_path: admin/server/asterisk
asterisk_min_version: 12
default_user_registration_mode: private
default_tts_engine: flite
post_install_management_mode: webui

## Business logic

`caspbx` is centered on Asterisk. The application is responsible for configuring, orchestrating, and presenting the telephony features of an Asterisk-driven deployment through one integrated interface, with Asterisk and compatible telephony, fax, messaging, and media subsystems acting as the underlying execution layer where required.

The primary product goal is a fully fledged, feature-rich Asterisk Web UI. After installation and initial bootstrap, an administrator should be able to perform normal platform, tenant, user, and telephony administration from the Web UI without needing to return to the terminal for routine operations.

Replacement model:

- Replace the many separate Web UIs commonly used around Asterisk with one integrated platform
- Replace the many supporting telephony helper apps commonly needed for fax, modem, operator, contact-center, reporting, media, and user-control workflows with built-in product functionality wherever practical
- Treat supporting system components as subordinate infrastructure, not as separate administrator-facing products
- Own the full lifecycle of telephony administration, user communications, operational control, and backend orchestration inside one product
- Eliminate the expectation that administrators need to learn and juggle numerous separate web panels to operate one communications system

Minimum replacement parity rule:

- For every named product surface `caspbx` is replacing, the minimum acceptable bar is 1:1 feature and capability parity
- This applies to the actual tasks users and administrators must be able to accomplish, not just loose conceptual coverage
- If an existing product UI exposes a practical feature, workflow, screen, control, report, routing option, provisioning option, operational action, or user-facing action that matters in real deployments, `caspbx` should match it at minimum
- The goal is not merely "similar category coverage"; the goal is to be able to replace those products without administrators losing expected capabilities
- `caspbx` is expected to exceed that baseline where it improves usability, integration, security, multitenancy, or operational coherence
- The product may unify, simplify, reorganize, or redesign routing and UX flows, but it must not reduce the underlying capability set below the minimum parity bar
- For fax, operator, user-portal, call-center, conferencing, and PBX admin surfaces alike, parity means administrators and end users can accomplish the same real work from `caspbx`, even if the navigation, route structure, and UI organization are improved

Deployment assumptions and external dependencies:

- Asterisk is the core communications engine and is always part of the deployment
- The product may rely on the packages, libraries, codecs, channel drivers, and supporting system components required for a full supported Asterisk installation and for supported telephony features
- SMTP delivery is an expected deployment dependency for notifications, password reset flows, invites, voicemail delivery, fax delivery, and related outbound mail features; supported deployment choices include common MTAs such as Postfix, Sendmail, and Exim4
- SIP trunks, carriers, gateways, devices, phones, browsers, storage, and DNS/domain infrastructure are expected deployment inputs rather than features replaced by `caspbx`
- `caspbx` owns orchestration, configuration, policy, UX, and management workflows; it does not replace the underlying need for telephony carriers, network connectivity, certificates, or external mail transport

Required installed components beyond `caspbx` itself:

1. **Core communications dependency**
    - A full supported Asterisk installation with the modules, codecs, applications, channel drivers, protocol support, and runtime services needed for the enabled feature set

2. **Outbound and system mail delivery**
    - A working SMTP path for transactional and telephony-related mail
    - This may be provided by a local MTA such as Postfix, Sendmail, or Exim4, or by a properly configured relay path exposed to the host

3. **Audio and media processing tooling**
    - System-level audio tooling required to normalize, transcode, inspect, trim, convert, and prepare prompts, recordings, music on hold, and other telephony media assets
    - Media support required to produce the audio formats expected by Asterisk and endpoint/webphone experiences

4. **Document and fax processing tooling**
    - System-level document/image tooling required to prepare, convert, render, inspect, and deliver fax-related documents and attachments
    - PDF/image/TIFF-style processing support sufficient for inbound fax handling, outbound fax preparation, previews, and mail/fax workflows

5. **Security and TLS support**
    - System certificate trust, crypto, TLS, and key-management support required by secure web access, secure transport configuration, certificate workflows, and outbound integrations

6. **Host runtime and scheduling support**
    - Standard OS facilities for process supervision, timers/scheduling, file permissions, networking, DNS resolution, logging, and persistent storage
    - Time synchronization support suitable for call records, scheduling, wakeup/reminder behavior, logs, and certificate validity

7. **Storage and filesystem support**
    - Reliable persistent storage for configuration state, generated configuration, recordings, voicemail, prompts, documents, fax artifacts, backups, and audit/event data

8. **Feature-dependent support packages**
    - Any host packages directly required by enabled Asterisk capabilities such as conferencing, media translation, device provisioning, presence/messaging support, browser calling support, or fax/document flows
    - These are deployment dependencies, not separate administrator-facing products

Minimal external dependency philosophy:

- The product should intentionally rely on as few separate administrator-facing systems as practical
- The core expected external stack is:
  - Asterisk and the packages needed for the enabled Asterisk feature set
  - SMTP delivery infrastructure
  - Standard OS/runtime facilities
- Additional installed components should primarily be low-level support packages, libraries, codecs, conversion tools, and runtime services rather than separate telephony management products
- If an additional package is needed, it should exist to support Asterisk or `caspbx`, not to reintroduce another primary PBX/fax/operator/admin surface

What should NOT be required as separate primary products:

- a separate PBX administration suite
- a separate end-user communications portal
- a separate operator panel product
- a separate call-center management UI
- a separate fax administration product
- a separate IVR/routing web UI
- a separate day-to-day backend telephony admin panel
- a separate collection of module storefronts, paid unlock systems, or feature packs

What the product must own:

- telephony configuration
- call routing logic
- dialplan intent and rendered behavior
- extension and device lifecycle
- user communications controls
- operator and supervisor operations
- call-center operations and reporting
- IVR, prompts, media, and TTS workflows
- voicemail and recordings management
- fax sending, receiving, routing, storage, and delivery workflows
- messaging and presence where supported by the stack
- conferencing administration and participant controls
- reminders, wakeup, schedules, and time-based automations
- multitenancy, organizations, domains, auth, permissions, and auditability
- backend stack visibility, health, configuration, and routine administration
- the full operational loop from configuration to live operation to historical review

Core product direction:

- Provide a complete frontend for Asterisk systems managed by the application
- Manage and integrate the backend telephony/fax stack used by the platform, including Asterisk and the compatible supporting services needed for the current deployment
- Provide full XMPP support wherever Asterisk exposes XMPP-related functionality
- Provide full PBX management coverage, including all major feature areas expected from a complete Asterisk administration suite
- Include the entire PBX and communications management surface as part of the core product rather than through separately installed plugins or commercial add-ons
- Integrate fax workflows as first-class platform functionality, whether delivered through Asterisk-native paths or compatible backend fax/modem subsystems where required
- Provide full IVR support, including graphical IVR building, nested menus, schedules, recordings, keypress routing, queue routing, time conditions, and destination selection
- Provide TTS support for prompts and telephony flows, using Flite as the sane default local engine and allowing commercially friendly external TTS providers where appropriate
- Provide live operator panel and live call control surfaces
- Provide reminders, wakeup features, directory/contact tooling, and conference management
- Provide call center functionality for queues, agents, monitoring, and operational workflows
- Provide fax-to-mail and mail-to-fax workflows
- Provide a full end-user communications control panel, including a full webphone experience
- Support music on hold management, including both local and remote sources
- Support both single-tenant and hosted multi-tenant operation from day one
- Support provider/platform administration, tenant/customer management, user/extension management, and organization support inside tenants where needed
- Support custom domains for tenants or organizations from day one
- Expose the rest of the Asterisk-driven platform features needed to make the application a complete communications frontend rather than a narrow single-purpose tool
- If Asterisk has a configurable or usable feature in supported versions, `caspbx` should expose configuration and management for it
- Aim for an exhaustive built-in Asterisk communications platform at the feature and workflow level while allowing implementation differences under the hood
- Treat minimum 1:1 parity with each replaced product surface as the floor, not the stretch goal
- Treat all named feature areas as mandatory for the first shippable release rather than deferring them to later milestone releases
- Use `/admin` as the primary admin path and expose full stack-specific Asterisk administration under `/admin/server/asterisk`
- Do not use a plugin, module marketplace, or feature-pack system for core platform capabilities
- Do not gate functionality behind paid tiers, module unlocks, or separately licensed feature packs

Product principles:

- One product, one admin surface, one operational model
- Built-in capability over bolt-on modules
- Replace toolchains with workflows
- Web UI first for routine operations
- Secure by default and authenticated by default
- Provider-grade and tenant-aware from day one
- End-user, operator, and admin experiences are all first-class
- Parity first, then improvement
- Capability-driven exposure over dead or unusable UI
- If the underlying Asterisk stack supports it safely, the product should expose it coherently
- Configuration intent should be higher level than raw config-file editing
- The product should abstract backend complexity rather than expose it unnecessarily

Authentication and access defaults:

- Use authentication by default for all administrative, operational, tenant, organization, and end-user telephony functions
- Keep public unauthenticated routes limited to what is operationally necessary, such as login, logout, password reset, invite acceptance, domain verification flows, and explicitly public-facing pages
- Default regular user registration mode to `private` / invite-only unless an administrator explicitly enables broader registration
- Put the primary administrative surface behind `/{admin_path}`
- Put full backend-stack administration behind `/{asterisk_admin_path}`
- Require elevated authenticated roles for backend-service configuration such as Asterisk core settings, fax/modem subsystem settings, messaging backend configuration, and system-level telephony integrations
- Require authenticated end-user access for the communications control panel, webphone, voicemail, personal fax, contacts, conferencing participation controls, and personal communications settings

Role model and scope boundaries:

- **Platform / Server Admin**: full-system control across the deployment, including global config, backend integrations, licensing-free feature management, updates, backups, and system-wide telephony configuration
- **Tenant / Customer Admin**: manages one tenant/customer scope, including users, extensions, devices, queues, IVRs, recordings, conferencing, tenant fax workflows, and tenant-level settings
- **Organization Admin**: manages org-scoped resources inside a tenant when organizations are enabled for that tenant
- **Supervisor / Operator**: manages operational surfaces such as call queues, live dashboards, reminders, operator panels, reporting, and call center workflows
- **Agent / Staff User**: uses call-center, queue, communications, and limited operational tools assigned by role
- **End User / Extension User**: uses the communications control panel, webphone, voicemail, presence, contacts, conferencing, messaging, personal forwarding, and personal communication features

Web UI administration principles:

- The Web UI must expose all routine configuration and operations needed after installation
- Terminal usage should be reserved for initial installation, host operating-system work, or exceptional disaster-recovery situations
- Anything the backend stack expects administrators to manage regularly should have a Web UI workflow
- All backend-service configuration that is safe and relevant to expose should be manageable in the Web UI, especially under `/{asterisk_admin_path}`
- The Web UI must include guided setup, validation, previews, dependency checks, and error messaging so administrators are not forced into manual config-file editing for normal workflows

Asterisk integration model:

- `caspbx` is the system-of-record for PBX intent, tenant-scoped telephony configuration, user communications settings, and operational policy; Asterisk runtime configuration is generated and synchronized from `caspbx`, not edited manually as the primary workflow
- `caspbx` must compile high-level objects such as tenants, extensions, trunks, routes, queues, IVRs, conferences, prompts, fax endpoints, and permissions into the concrete Asterisk configuration, dialplan, and runtime actions required by the installed version
- `caspbx` must hook into Asterisk through multiple integration planes rather than a single API:
  - **Configuration plane**: generate, validate, write, version, and safely apply Asterisk-related configuration artifacts and supporting service configuration
  - **Control plane**: issue runtime actions such as originate, hangup, transfer, queue control, conference control, endpoint actions, reloads, and module/service lifecycle operations
  - **Event plane**: ingest call events, endpoint state, queue activity, conference activity, fax state, voicemail state, and backend-service health into the platform for dashboards, automation, and audit trails
  - **Media plane**: manage prompts, music on hold, voicemail assets, call recordings, fax documents, TTS outputs, and related media lifecycle operations
  - **Provisioning plane**: produce endpoint/device configuration, tenant-aware defaults, and compatibility-specific templates for the active deployment
- `caspbx` should use the appropriate Asterisk and system integration mechanisms for the job, including configuration rendering, safe reload/apply workflows, CLI/module interactions where necessary, management/control interfaces, event streams, spool or job directories, and runtime metadata stores
- Dialplan generation is a core responsibility of `caspbx`; administrators should manage call flow intent in the UI, while the platform renders and applies the corresponding dialplan and supporting configuration
- Capability detection is required at install time and continuously relevant after upgrades; `caspbx` must detect which channels, modules, applications, codecs, fax paths, conferencing features, and messaging/presence features are available before exposing or applying related features
- `caspbx` must own realtime operational state for the UI by consuming Asterisk and service events into a normalized internal model instead of forcing the UI to scrape raw command output
- `caspbx` must treat supporting telephony apps and helper daemons as internal implementation details when they are needed; administrators interact with `caspbx`, not with separate operator, fax, modem, or event-console products as primary tools
- Where a legacy helper component is still needed for compatibility or transport reasons, `caspbx` should install, configure, monitor, and abstract it behind one platform UX so the product still functions as the replacement surface
- All apply/reload/restart workflows must support preview, validation, diff visibility where useful, rollback-safe behavior where practical, and clear operator messaging

Boundaries of responsibility:

- `caspbx` owns the communications application layer and administrative UX
- Asterisk and required supporting services perform the underlying call, media, messaging, or fax work
- `caspbx` should replace separate administrator-facing products, but it does not replace the underlying existence of required telephony infrastructure
- When a capability requires an external transport or service, `caspbx` should configure and manage the relationship rather than pretending the dependency does not exist
- When a capability can be implemented fully inside the product plus the Asterisk stack, that integrated path is preferred over exposing another standalone tool

Asterisk compatibility and feature exposure policy:

- `caspbx` supports Asterisk version 12 and newer
- Feature support is capability-driven across Asterisk versions: if a supported installed version exposes a feature and it can be configured safely, `caspbx` should surface it
- Version-specific differences should be handled by capability detection, compatibility layers, and clear UI messaging rather than by requiring terminal inspection
- If a feature exists in newer Asterisk versions but not the current deployment version, the UI should show it as unavailable or limited with an explanation rather than pretending it exists
- If multiple backend implementations are possible for a feature, the UI should guide the administrator toward the supported combination for the current deployment
- If a hardware-backed or integration-backed feature is not present on the current deployment, `caspbx` should hide or suppress the corresponding admin UI rather than showing dead configuration surfaces
- When a previously unavailable capability becomes available, the relevant UI should appear automatically with the correct role gating, validation, and explanatory messaging
- Examples include DAHDI-backed hardware interfaces, specific codec-dependent features, optional fax/document paths, and backend integrations that are not installed or not currently usable

Approved project-specific deviation from AI.md PART 34:

- Although AI.md PART 34 defaults multi-user registration to `public` when enabled, `caspbx` uses `private` / invite-only as the project default by explicit approval for this project
- Public self-registration may still be enabled later by administrators through configuration, but invite-driven onboarding is the sane default

Primary feature domains:

1. PBX and telephony administration
    - Extension, trunk, route, dialplan, device, queue, IVR, ring group, conference, voicemail, XMPP, music on hold, and feature management
    - Full server administration comparable to a complete PBX management suite
    - Operational views for active calls, registration state, endpoints, and telephony health
    - Full Asterisk stack administration under `/admin/server/asterisk`, including fax/modem backend management and related supporting-service administration
    - Web UI workflows for everything commonly handled through separate PBX administration products or helper tools
    - Built-in administration for the full PBX/communications surface rather than separately installable module packs
    - Coverage should include the broad feature surface administrators expect from a complete Asterisk management product, including the depth and breadth typically associated with a 100+ feature/module PBX environment

2. Endpoint, device, and provisioning management
    - Phone, endpoint, device, codec, transport, registration, template, and provisioning administration
    - Device assignment, reset, replacement, status, and compatibility management
    - Tenant-aware defaults and deployment-aware provisioning output
    - Endpoint diagnostics, registration troubleshooting, and operational state visibility

3. User communications experience
    - User control panel for everyday calling features
    - Full browser-based webphone
    - Access to voicemail, presence/status, call handling controls, contacts, conferencing, and personal communication settings
    - Personal forwarding, do not disturb, call recording access where allowed, follow-me, device preferences, and message/event visibility
    - Invite-driven onboarding and self-service features consistent with role and tenant policy

4. Fax and document workflows
    - Send and receive fax management
    - Fax inbox/outbox/history
    - Fax-to-mail and mail-to-fax flows
    - Support for deployments using Asterisk-native fax flows or compatible backend fax/modem subsystems required by the stack
    - DID routing, tenant routing, mailbox routing, archival controls, retry state, delivery state, and operator visibility
    - Cover the practical administrator and user workflows that would otherwise require separate fax/modem administration tools

5. Realtime messaging and presence
    - XMPP configuration and operational support where provided by supported Asterisk versions
    - Presence, messaging-related administration, and integration points tied to the Asterisk stack
    - Status, routing, policy, and identity controls needed for messaging-capable deployments

6. IVR, prompts, media, and routing automation
    - Graphical IVR builder with reusable menus, nested trees, and destination actions
    - Prompt library for uploaded audio, recorded audio, generated TTS audio, and reusable media assets
    - TTS support using Flite by default, with optional support for commercially friendly external APIs where approved for use
    - Schedule-aware routing, holiday routing, time conditions, failover actions, and prompt preview/testing in the Web UI
    - Music on hold management with local and remote source support
    - Full call-flow building for common and advanced routing scenarios without routine dialplan hand editing
    - Recording management, playback policy, prompt versioning, and media assignment to features across the platform

7. Operator and contact center tools
    - Operator switchboard workflows
    - Queue and agent management
    - Call center reporting, monitoring, and real-time operational control
    - Wallboards, campaign-adjacent operational visibility, supervisor actions, queue events, and historical reporting
    - Live intervention workflows such as listen, whisper, barge, transfer, and queue state management where supported

8. Conferencing and collaboration
    - Conference room lifecycle management
    - Participant controls, moderation, access policy, recordings where supported, and scheduling where applicable
    - User-facing and admin-facing conference workflows

9. Reminders, wakeup, directories, and automation
    - Reminder scheduling, wakeup scheduling, retry behavior, delivery behavior, and status visibility
    - Directory/contact management for users, operators, and tenant/global contexts
    - Time-based automations and operational workflows tied to telephony behavior

10. Tenant-aware platform management
    - Multitenant hosting support alongside single-tenant deployment support
    - Provider/platform administration above tenant/customer boundaries
    - Separation of tenant data, settings, and operational controls
    - Tenant-scoped administration alongside server/platform administration
    - Optional organization layer inside tenants when required by the deployment model
    - Custom domain support for tenants or organizations
    - Per-tenant branding, routing scope, policy scope, and delegated administration

11. Platform administration and observability
    - Setup flows, health checks, audit trails, service status, capacity visibility, backups, updates, and operational diagnostics
    - Role/permission management across provider, tenant, organization, supervisor, agent, and end-user roles
    - Notification routing, outbound mail configuration, storage policy, retention policy, and compliance-aware operational controls
    - Replace the routine system-app hopping normally required to run an Asterisk communications environment

## Comprehensive scope statement

The product should be thought of as the full integrated answer for an Asterisk deployment's communications UX and administration, not as one piece of it. If an administrator would normally expect to open another PBX-related web interface, operator console, fax tool, call-center panel, media/routing tool, or day-to-day backend administration surface to keep the system running, `caspbx` should either:

1. provide that capability directly inside the product, or
2. fully abstract and manage the required helper subsystem so the administrator still works primarily inside `caspbx`

The practical result is that `caspbx` is expected to replace the experience of operating a communications stack through 7-12 separate Web UIs and numerous system tools. A deployment may still contain underlying packages and services, but the product should make them feel like one coherent platform.

This replacement expectation includes a minimum requirement that each replaced surface remain functionally whole inside `caspbx`. Replacing a product means matching its meaningful capability set and real-world workflows first, then integrating and improving them inside one platform.

Business invariants:

- Asterisk is the core communications engine for the platform
- Asterisk and related telephony, fax, messaging, and media infrastructure are backend infrastructure for the platform, while `caspbx` is the unified management and user-facing frontend
- The application is a complete telephony frontend, not just a thin configuration editor
- The project is intended to be free and full-featured rather than artificially limited
- The product ships with built-in capabilities rather than a plugin/module marketplace
- The product must support both administrator-facing and end-user-facing telephony workflows
- Routine post-install administration should be possible entirely from the Web UI
- Single-tenant and hosted multi-tenant operation are both first-class modes from day one
- Provider/platform admins, tenant/customer admins, and end users are distinct product roles
- Organizations are supported inside tenants where the deployment requires them
- Custom domains are part of the core product scope from day one
- PBX, fax, conferencing, call center, reminders, wakeup, directory, and user communication features belong in one integrated application
- XMPP and messaging-related features exposed by supported Asterisk versions belong in scope
- Full IVR support belongs in scope
- TTS-backed prompt generation belongs in scope, with Flite as the sane default engine
- Music on hold management, including local and remote sources, belongs in scope
- The product is expected to replace fragmented PBX, operator, fax, and user-control stacks with one integrated system
- The full communications feature surface is built into the platform rather than delivered as separately installed modules
- Minimum 1:1 parity with each replaced product surface is required; broader and better integrated behavior is the target above that floor
- The first shippable release includes all named feature areas rather than a reduced MVP
- Supported Asterisk compatibility starts at version 12 and extends through later supported versions
- If a supported Asterisk version exposes a configurable capability, `caspbx` should provide a way to configure and use it
- If a capability is not available in the current deployment, the product should not clutter the UI with unusable controls for it
- Administrative and operational surfaces should be protected by authentication unless there is a clear public use case
- Sane defaults should favor secure, authenticated, invite-driven access over unnecessary public exposure
- No feature may be hidden behind paid gating, premium unlocks, or optional commercial module packs
- `caspbx` is the primary control surface; any retained helper process or backend subsystem is subordinate to and managed by the platform
