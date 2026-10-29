# Improvements

## Interaction model: work in place

Most routine work now happens in place: Board and List expand the task, Calendar uses an inline
agenda on desktop and a seven day list on mobile, Notes autosaves, details use a non-modal panel or
the `/tasks/:id` route, and the Electron menu mirrors the shortcuts. The remaining gaps:

- The inline subtask checklist accepts a new step on Enter, but Tab does not indent it under the
  previous step.
- Verify by hand that the undo toast pauses on hover or focus, does not cover the focused row, and
  expires after at least 8 seconds.

## Assistant on Telegram or WhatsApp

The Telegram assistant now runs in the API binary over `getUpdates` long polling: allowlist by
user and chat, private chats only, text and voice, `find_task` and `move_task` with inline
confirmation, an in-memory 10-message history, and a reminder channel. Remaining work:

- WhatsApp Cloud API behind a shared inbound transport interface. The current bot is
  Telegram-specific. WhatsApp needs a Meta business account, a verified number, a public webhook,
  the 24 hour service window, and templates for proactive messages.
- Smoke-test raw OGG/Opus against the gateway. If it is rejected, convert to mono mp3 and add ffmpeg
  to the backend image.
- Anthropic has no audio input, so voice requires an OpenAI-compatible provider. Keep transcription
  behind `ai.Provider` so a swap is one config change.
- Run one live pass with a real bot token.
