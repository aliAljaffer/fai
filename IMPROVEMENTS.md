# Improvements

## Interaction model: work in place

Most routine work now happens in place: Board and List expand the task, Calendar uses an inline
agenda on desktop and a seven day list on mobile, Notes autosaves, details use a non-modal panel or
the `/tasks/:id` route, and the Electron menu mirrors the shortcuts. Steps can nest: Tab in the
inline checklist indents the next item under the previous one, and nested steps render in the
inline editor and the task detail panel. The undo toast waits 8 seconds, pauses on hover or focus,
and sits above the mobile tab bar. Remaining gaps:

- Check the undo toast against the focused row on a short viewport by hand. Placement is fixed,
  so a row near the bottom edge can still sit under it.

## Telegram assistant

The assistant runs in the API binary over `getUpdates` long polling: allowlist by user and chat,
private chats only, text and voice, `find_task` and `move_task` with inline confirmation, an
in-memory 10-message history, and a reminder channel. WhatsApp is out of scope. Remaining work:

- Smoke-test raw OGG/Opus against the gateway. If it is rejected, convert to mono mp3 and add ffmpeg
  to the backend image.
- Anthropic has no audio input, so voice requires an OpenAI-compatible provider. Keep transcription
  behind `ai.Provider` so a swap is one config change.
- Run one live pass with a real bot token.
