# Improvements

## Interaction model: work in place

Most routine work now happens in place: Board and List expand the task, Calendar uses an inline
agenda on desktop and a seven day list on mobile, Notes autosaves, details use a non-modal panel or
the `/tasks/:id` route, and the Electron menu mirrors the shortcuts. Steps can nest: Tab in the
inline checklist indents the next item under the previous one, and nested steps render in the
inline editor and the task detail panel. The undo toast waits 8 seconds, pauses on hover or focus,
sits above the mobile tab bar, and scrolls a focused row clear of itself when it opens and on every
focus change. Fixed placement no longer hides a row at the bottom of a short viewport.

## Telegram assistant

The assistant runs in the API binary over `getUpdates` long polling: allowlist by user and chat,
private chats only, text and voice, `find_task` and `move_task` with inline confirmation, an
in-memory 10-message history, and a reminder channel. Voice notes are sent as raw OGG/Opus first.
If the gateway rejects them, the backend transcodes to mono 16 kHz MP3 with the static ffmpeg
bundled in the image and retries. Transcription stays behind `ai.Provider`, so an OpenAI-compatible
provider is still required; Anthropic returns a clear error. WhatsApp is out of scope.

Remaining work:

- Run one live pass with a real bot token. Nothing else is open.
