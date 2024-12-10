# CHASE
## Certificate Hunting & Security Enumeration

Tool to find security issues for a given domain.
It checks for:
- security headers
- certificate best practices
- screenshots of the domain over time
- exposure of admin pages
- exposure of swagger endpoints

## Get started
```bash
git clone https://github.com/NorskHelsenett/chase.git
cd chase
code .
```

## Devcontainer

Open with VSCode Devcontainer support

## Setup OIDC (optional)
Create an `.env` file,
```bash
cat <<EOF > /api/.env
OIDC_ISSUER_URL=
OIDC_CLIENT_ID=
OIDC_CLIENT_SECRET=
OIDC_REDIRECT_URL=http://localhost:5173/api/callback
EOF
```

## Start debugging
F5 to start debugging golang
