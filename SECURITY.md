# Security

## Reporting a vulnerability

Report privately through GitHub: open the **Security** tab and choose
**Report a vulnerability**. Do not open a public issue.

Include the affected version, what you did, and what happened. Expect a first
response within a few days.

## Supported versions

Only the latest release and the `main` branch receive fixes.

## Deployment notes

- The backend fails closed: with auth enabled it refuses to start unless the
  OIDC issuer and audience are set. Never set `DISABLE_AUTH=true` outside local
  development.
- `AUTH_MODE=proxy-header` trusts `X-Forwarded-User`. Only use it when the
  backend is unreachable except through the proxy that sets that header.
- The calendar feed at `/api/calendar.ics` is authenticated by its URL token
  alone. Treat that URL as a credential.
- Published container images and desktop installers are built from this
  repository by GitHub Actions.
