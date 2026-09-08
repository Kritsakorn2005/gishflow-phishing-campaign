# Deploying GishFlow

This guide puts GishFlow on a public server so real recipients can receive
messages and interact with the phishing pages. The phishing server is served
over HTTPS by Caddy (automatic Let's Encrypt); the admin panel stays private.

> ⚠️ **Authorization first.** Running a phishing simulation against real people
> requires documented consent/authorization (advisor, faculty, or an ethics/IRB
> approval). Do not launch campaigns at real recipients without it. GishFlow's
> teachable-moment page helps, but it does not replace approval.

## Prerequisites
- A Linux server (e.g. Ubuntu 22.04) with a **public IP**, ports **80** and
  **443** open.
- A **domain/subdomain** for the phishing server (e.g. `phish.example.com`).
- Docker + the Compose plugin installed.

## 1. DNS
Create an **A record** for your phishing domain pointing at the server's IP:

    phish.example.com   A   <SERVER_PUBLIC_IP>

Wait for it to resolve (`dig +short phish.example.com`).

## 2. Install Docker (Ubuntu)
```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER   # log out/in afterwards
```

## 3. Get the code and configure
```bash
git clone https://github.com/Kritsakorn2005/gishflow-phishing-campaign.git
cd gishflow-phishing-campaign/deploy
cp .env.example .env
nano .env        # set PHISH_DOMAIN, CONTACT_ADDRESS, and AI_* if used
```

## 4. Launch
```bash
docker compose up -d --build
```
Caddy will obtain a TLS certificate for `PHISH_DOMAIN` automatically (this needs
DNS to resolve and ports 80/443 reachable). Check it:
```bash
docker compose logs -f caddy      # look for a successful certificate
curl -I https://phish.example.com # should return an HTTP response
```

## 5. Reach the admin panel (private)
The admin panel is bound to localhost on the server. Tunnel to it over SSH from
your laptop:
```bash
ssh -L 3333:localhost:3333 user@<SERVER_PUBLIC_IP>
```
Then open <http://localhost:3333>. First login uses the password printed once in
the app logs:
```bash
docker compose logs gishflow | grep -i "Please login with the username"
```
Change the password immediately.

## 6. Wire up a campaign
- **Campaign URL:** set it to `https://phish.example.com` so links, tracking,
  QR codes, and landing pages resolve over HTTPS.
- **LINE webhook:** in the LINE console set the webhook to
  `https://phish.example.com/line/webhook` (no more ngrok). Add your channel
  secret to the LINE sending profile to enforce signature verification.
- **Email deliverability:** use a real SMTP/relay and set up **SPF, DKIM, and
  DMARC** for the sending domain, or messages will land in spam.

## Operations
- **Data** lives in the `gishflow-data` volume (sqlite). Back it up:
  ```bash
  docker run --rm -v deploy_gishflow-data:/d -v $PWD:/b alpine \
    tar czf /b/gishflow-backup.tgz -C /d .
  ```
- **Update:** `git pull && docker compose up -d --build`
- **Logs:** `docker compose logs -f gishflow`
- **Stop:** `docker compose down` (data volume is kept)

## Notes
- For larger tests, switch to MySQL: set `DB_NAME=mysql` and `DB_FILE_PATH` to a
  DSN, and add a `mysql` service (see the project README).
- ngrok/Cloudflare Tunnel remain fine for quick pilots; this compose is for a
  stable field test on your own domain.
