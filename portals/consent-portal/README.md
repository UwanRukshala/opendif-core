# Consent Portal

A citizen-facing React application that allows data owners to view, approve, or deny data access requests.

## Overview

The Consent Portal is the interface where citizens (data owners) interact with OpenDIF to manage their data consents. It is typically accessed via a redirect from a data consumer application when consent is required.

**Technology**: React + TypeScript + TailwindCSS + Vite

## Features

- **Consent Review** - View details of data access requests (who, what, why)
- **Approval/Denial** - Grant or deny access to requested data
- **Consent Management** - View and revoke previously granted consents
- **Secure Authentication** - Integration with IdP for user authentication

## Quick Start

### Prerequisites

- Node.js 18+
- npm 9+

### Run the Application

```bash
# Install dependencies
npm install

# Create your runtime config (see Configuration below)
cp public/config.example.js public/config.js

# Run in development mode
npm run dev
```

The application will be available at **`http://localhost:3002`**. The dev server
is pinned to port **3002** (`strictPort: true`) to match the eSignet redirect URI.

## Configuration

The portal is configured at **runtime**, not at build time. `public/config.js`
is loaded as a plain `<script>` in `index.html` and exposed on `window.configs`,
so the same build can be pointed at different environments just by swapping this
file — no rebuild required.

`public/config.js` is gitignored. Create it by copying the template:

```bash
cp public/config.example.js public/config.js
```

Then set the values in `public/config.js`:

| Key                     | Description                                                                                                 |
|-------------------------|-------------------------------------------------------------------------------------------------------------|
| `consentEngineUrl`      | Base URL of the Consent Engine API (e.g. `http://localhost:8081/api/v1`)                                    |
| `idpClientId`           | OAuth2 client ID registered with eSignet (e.g. `ndx`)                                                       |
| `idpBaseUrl`            | eSignet UI base URL; OIDC discovery at `idpBaseUrl/.well-known/openid-configuration`                        |
| `idpScope`              | Space-separated OAuth2 scopes (e.g. `openid profile email`)                                                 |
| `idpSignInRedirectUrl`  | Post-login redirect URL (must be registered with eSignet)                                                   |
| `idpSignOutRedirectUrl` | Post-logout redirect URL (must be registered with eSignet)                                                  |

Token exchange uses eSignet `private_key_jwt` and is handled by the Consent Engine
(`POST /api/v1/auth/token`). Configure the backend env vars below.

### SLUDI / eSignet (dev example)

**Consent portal** (`public/config.js`):

```javascript
window.configs = {
  consentEngineUrl: 'http://localhost:8081/api/v1',
  idpClientId: 'ndx',
  idpBaseUrl: 'https://esignet.dev.digieconcenter.gov.lk',
  idpScope: 'openid profile email',
  idpSignInRedirectUrl: 'http://localhost:3002',
  idpSignOutRedirectUrl: 'http://localhost:3002',
};
```

**Consent Engine** (`.env`):

| Env var | Value |
|---------|-------|
| `IDP_CLIENT_ID` | `ndx` |
| `IDP_ISSUER` | `https://esignet.dev.digieconcenter.gov.lk` |
| `IDP_AUDIENCE` | `https://gateway.dev.digieconcenter.gov.lk/sludi-oidc/v1.0.1/oauth/v2/token` |
| `IDP_JWKS_URL` | `https://esignet.dev.digieconcenter.gov.lk/.well-known/jwks.json` |
| `IDP_TOKEN_ENDPOINT` | `https://esignet.dev.digieconcenter.gov.lk/v1/esignet/oauth/v2/token` |
| `IDP_PRIVATE_KEY` | RSA private key PEM registered with eSignet for client `ndx` |
| `CONSENT_PORTAL_URL` | `http://localhost:3002` |

## Testing Guide

### End-to-End Flow

1. **Start Backend Services**: Ensure Consent Engine is running on port 8081.
2. **Generate Consent Request**:
   - Use Postman or curl to create a consent request in Consent Engine.
   - Copy the `consent_id` from the response.
3. **Access Portal**:
   - Navigate to `http://localhost:3002/?consentId={consent_id}`
   - Log in if required.
   - Review and act on the consent request.