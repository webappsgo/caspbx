# Setup Tasks for caspbx

## High (P1) - Project definition

- [x] Replace the placeholder text in `IDEA.md` with the real project description and business logic
- [ ] Add the remaining confirmed project variables still missing from `IDEA.md` (for example `official_site` and maintainer details)
- [x] Confirm the exact tenant model and map it to optional PARTS 34-36 before changing any OPTIONAL → REQUIRED markers
- [x] Define how closely `caspbx` must match FreePBX/UCP/FOP2/AvantFax module parity versus where the product is allowed to differ
- [x] Confirm the first-class backend matrix: Asterisk + TFRP + HylaFAX+ + IAXModem, with Asterisk-native fax also supported
- [x] Define the first-release scope for call center, conferencing, reminders, wakeup, asterdex, fax2mail, mail2fax, and webphone features
- [x] Update the per-project AI.md to mark PARTS 34-36 as required for this project and keep IDEA.md aligned with that decision
- [x] Define the exact compatibility/support policy for Asterisk 12+ so feature behavior is clear across version differences
- [x] Define sane default auth boundaries and post-install Web UI expectations
- [x] Add IVR, TTS, and no-terminal-after-install product requirements to the spec

## Medium (P2) - Bootstrap

- [ ] Translate the product definition into implementation phases and architecture decisions
- [ ] Create the remaining project files and directories required for a real implementation (`README.md`, `LICENSE.md`, `src/`, `docker/`, `tests/`, and related files)
- [ ] Bring the repository contents into compliance with the project structure defined in `AI.md`
