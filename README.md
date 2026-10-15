# GishFlow

**A multi-channel phishing-simulation and security-awareness platform**, built as an
extension of the open-source [Gophish](https://github.com/gophish/gophish) toolkit.

GishFlow adds multi-channel delivery, AI-assisted content, a NIST-Phish-Scale–based
difficulty rating, a behavioral risk indicator, and a teachable-moment page on top of
Gophish's campaign engine. It was developed as a research/education project at the
College of Computing, Khon Kaen University.

> ⚠️ **Authorized use only.** This is a phishing-*simulation* tool for security
> awareness training and research. Running campaigns against real people requires
> documented consent/authorization (advisor, faculty, or an ethics/IRB approval).
> Do not use it to attack anyone. You are responsible for how you use it.

---

## Features

- **Multi-channel delivery** — Email, SMS (self-hosted Android gateway), LINE
  (Official Account webhook that captures userId automatically), and QR code.
- **AI-assisted content** — generates simulated pretext and quiz drafts via any
  OpenAI/Anthropic-compatible endpoint (e.g. KKU IntelSphere); an admin reviews
  before use.
- **Difficulty rating** — detection difficulty adapted from the NIST Phish Scale.
- **Behavioral risk indicator** — per-user, time-decayed risk score with trend.
- **Teachable moment** — instant awareness page with a short quiz and completion
  tracking.
- **Deliverability** — Amazon SES with SPF/DKIM/DMARC; Caddy automatic HTTPS for the
  public phishing server; the admin panel stays private (localhost / SSH tunnel).

## Requirements

- **Docker Desktop** — all build/run happens in a container (the SQLite driver needs
  CGO, so the app is built inside `golang:1.15.2`; you do not need Go or Node locally).
- **Git**

## Quick start (local development)

```bash
# 1) Make your own local config (copied from the default; run-config.json is gitignored)
cp config.json run-config.json

# 2) Create a network and run MailHog (catches outgoing email during dev)
docker network create gishflow-net
docker run -d --name mailhog --network gishflow-net -p 8025:8025 mailhog/mailhog

# 3) Build + run the app (mounts the source at /app, keeps the DB in a volume)
docker run -d --name gishflow --network gishflow-net \
  -p 3333:3333 -p 8080:80 \
  -v "$(pwd)":/app -v gishflow-data:/data -v gishflow-gocache:/go \
  -v "$(pwd)/run-config.json":/run/run-config.json \
  -w /app golang:1.15.2 \
  sh -c "go build -o /tmp/gophish . && exec /tmp/gophish --config /run/run-config.json"

# 4) Watch the log — on first run it prints the admin username and password
docker logs -f gishflow
```

- **Admin UI:** <http://localhost:3333> (log in with the credentials from the first-run log, then change the password)
- **Phishing server:** <http://localhost:8080>
- **MailHog inbox:** <http://localhost:8025>

On Windows, if `$(pwd)` misbehaves, use the full path, e.g. `-v C:\path\to\gophish:/app`.

Full developer guide (building the JS/React assets, migrations, CI notes):
see **[HANDOFF.md](HANDOFF.md)**.

## Deployment

To put GishFlow on a public server (Docker Compose + Caddy auto-HTTPS), see
**[deploy/DEPLOY.md](deploy/DEPLOY.md)**.

## 🔑 Credentials — read this before you configure anything

**No real credentials are stored in this repository, and none should ever be committed.**
All external-service secrets belong only in your local, gitignored config or in the UI.
When you clone or fork this project, set up **your own** credentials:

| Service | Where you set it |
|---|---|
| AI (pretext/quiz) | the `ai` block in `run-config.json` — `base_url`, `api_key`, `model`, `provider` |
| Email (Amazon SES) | an Email sending profile (SMTP host/user/pass); configure SPF/DKIM/DMARC in DNS |
| LINE | a LINE sending profile + channel access token; webhook → `/line/webhook` |
| SMS | an SMS sending profile + your Android gateway URL |
| Public URL | Cloudflare Tunnel or ngrok to `localhost:8080`, or a real domain (see DEPLOY.md) |

`run-config.json`, `deploy/.env`, the SQLite database, and TLS keys are all listed in
[`.gitignore`](.gitignore). During dev you can send mail through MailHog with no SES
account (sending-profile host `mailhog:1025`).

## Credits

GishFlow is built on **[Gophish](https://github.com/gophish/gophish)** by Jordan Wright,
an open-source phishing framework released under the MIT License. The multi-channel
delivery, AI content, Phish Scale rating, behavioral risk indicator, and teachable-moment
features are additions made for this project.

## License

GishFlow inherits Gophish's **MIT License** — see [LICENSE](LICENSE).

```
Copyright (c) 2013-2020 Jordan Wright (Gophish)

Permission is hereby granted, free of charge, to any person obtaining a copy of this
software and associated documentation files (the "Software"), to deal in the Software
without restriction, including without limitation the rights to use, copy, modify,
merge, publish, distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to the conditions in
the LICENSE file.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND.
```
