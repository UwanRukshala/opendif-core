// Runtime configuration injected via public/config.js and exposed on
// `window.configs`. See public/config.example.js for the template.
export interface AppConfig {
  // Base URL of the Consent Engine API.
  consentEngineUrl: string;
  // SLUDI / eSignet OIDC settings (authorization uses eSignet UI discovery).
  idpBaseUrl: string;
  idpClientId: string;
  idpScope: string;
  // OAuth2 redirect URLs (must be registered with eSignet).
  idpSignInRedirectUrl: string;
  idpSignOutRedirectUrl: string;
}

declare global {
  interface Window {
    configs: AppConfig;
  }
}
