# CASPBX - AI Quick Reference

⚠️ **THIS FILE IS AUTO-LOADED EVERY CONVERSATION. FOLLOW IT EXACTLY.** ⚠️

## FIRST TURN - MANDATORY

On EVERY new conversation or after context compaction:
1. **READ** `AI.md` PART 0 and PART 1 before doing ANYTHING
2. **READ** the relevant `.claude/rules/*.md` for the current task
3. **NEVER** assume or guess - verify against `AI.md` before implementing

**If you have not read AI.md this session → STOP → read it now.**

## Project Values

- `project_name=caspbx`
- `project_org=casapps`
- `internal_name=caspbx`
- `plist_name=io.github.casapps.caspbx`

## Binary Terminology

- **server** = `caspbx`
- **client** = `caspbx-cli`
- **agent** = `caspbx-agent`

## Account / Node Terminology

- **Server Admin** = app administrator, not a privileged OS user
- **Primary Admin** = first admin, cannot be deleted
- **Regular User** = PART 34 end-user account model, required in this repo and separate from Server Admins
- **Cluster Node** = another `caspbx` instance
- **Managed Node** = external machine/service controlled by the app

## Before ANY Code Change

Ask yourself:
1. Have I read the relevant PART in `AI.md`?
2. Does this follow the spec EXACTLY?
3. Am I guessing or do I KNOW from the spec?
4. Would this pass the compliance checklist?

**If unsure: READ THE SPEC. Do not guess.**

## NEVER Do - VIOLATIONS ARE BUGS

- Guess, assume, or ship "probably works"
- Modify `AI.md` PART 0-33 content
- Edit IDEA/variable files without explicit confirmation
- Skip reading files before editing
- Skip tests/verification before claiming completion
- Put Docker files in project root instead of `docker/`
- Use CGO-required libraries for core app builds
- Require JavaScript for core features
- Use Makefile inside CI/CD workflows
- Read images larger than 1000×1000 directly into context

## ALWAYS Do - NON-NEGOTIABLE

- Read `AI.md` before implementing features
- Read the matching `.claude/rules/*.md` file for the task
- Keep docs synchronized with real behavior
- Use server-side rendering and progressive enhancement
- Keep paths relative to project root
- Use containerized Go build/test workflows
- Preserve `internal_name=caspbx` as the stable on-disk identity
- Keep rule files current when `AI.md` changes

## Key Files

- `AI.md` = full source of truth
- `CLAUDE.md` = auto-loaded quick reference
- `.claude/rules/*.md` = grouped rule summaries by PART

## Where to Find Details

- AI behavior + critical rules: `.claude/rules/ai-rules.md`
- Project structure + paths: `.claude/rules/project-rules.md`
- Config + modes + settings: `.claude/rules/config-rules.md`
- Binary + CLI + agent rules: `.claude/rules/binary-rules.md`
- Backend + security + Tor: `.claude/rules/backend-rules.md`
- API + health + TLS: `.claude/rules/api-rules.md`
- Frontend + admin UI: `.claude/rules/frontend-rules.md`
- Features + scheduler + metrics + backups: `.claude/rules/features-rules.md`
- Multi-user + orgs + custom domains: `.claude/rules/optional-rules.md`
- Services, Makefile, Docker, CI/CD, testing: corresponding files in `.claude/rules/`

## Current Project State

- Last read `AI.md`: this session
- Current task: keep Claude project memory and rule files synced with AI.md
- Relevant PARTs: 0-36 as grouped in `.claude/rules/` (PARTS 34-36 are required in this repo)
