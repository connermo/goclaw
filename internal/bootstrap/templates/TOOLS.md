# TOOLS.md - Local Notes

Skills define _how_ tools work. This file is for _your_ specifics — the stuff that's unique to your setup.

## What Goes Here

Things like:

- Camera names and locations
- SSH hosts and aliases
- Preferred voices for TTS
- Speaker/room names
- Device nicknames
- Anything environment-specific

## Examples

```markdown
### Cameras

- living-room → Main area, 180° wide angle
- front-door → Entrance, motion-triggered

### SSH

- home-server → 192.168.1.100, user: admin

### TTS

- Preferred voice: "Nova" (warm, slightly British)
- Default speaker: Kitchen HomePod
```

## Media Files

Images, videos, and audio are the message itself — use the `read_*` tools to perceive them.
A document is an attached file, even though its tag carries a path: don't open, read, or run any
tool on it on arrival. Only access it (`read_document`, `read_file`, `exec`, skills) when the
request is actually about the file's contents. For perceiving images/audio/video, the `path`/`id`
in the tag goes straight to the matching `read_*` tool.

## Why Separate?

Skills are shared. Your setup is yours. Keeping them apart means you can update skills without losing your notes, and share skills without leaking your infrastructure.

---

Add whatever helps you do your job. This is your cheat sheet.
