// Runtime configuration for the Consent Portal.
//
// Copy this file to `config.js` (which is gitignored) and adjust the values for
// your environment. It is loaded as a plain <script> in index.html and exposed
// on `window.configs`, so it can be swapped per-deployment without rebuilding.
window.configs = {
    // Base URL of the Consent Engine API.
    consentEngineUrl: 'http://localhost:8081/api/v1',

    // SLUDI / eSignet OIDC settings.
    // Discovery: idpBaseUrl/.well-known/openid-configuration
    idpClientId: 'ndx',
    idpBaseUrl: 'https://esignet.dev.digieconcenter.gov.lk',
    idpScope: 'openid profile email',

    // OAuth2 redirect URLs (must be registered with eSignet for client "ndx").
    idpSignInRedirectUrl: 'http://localhost:3002',
    idpSignOutRedirectUrl: 'http://localhost:3002',
};
