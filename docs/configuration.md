# Configuration

## Current Status

Runtime configuration files are intentionally **not committed** to the repository. The eventual server will generate its configuration on first run in the appropriate OS-specific configuration directory.

## Current Configuration Foundation

- configuration generated at runtime
- CLI flags override environment variables
- environment variables override config file values
- config file values override embedded defaults

## Implemented Foundation Areas

- server networking, request limits, compression, sessions, and rate limiting
- contact and maintenance settings
- custom-domain feature controls
- Asterisk admin foundation settings for compatibility floor, detection status, health status, capability toggles, and managed subsystem metadata

## Remaining Planned Areas

- server networking and listen addresses
- data, cache, log, backup, and PID paths
- tenant and auth settings
- telephony and backend integration settings
- capability-driven feature exposure
- SMTP and delivery settings
