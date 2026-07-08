import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import App from './App.tsx';
import { ConsentProvider } from './contexts/ConsentContext';
import { SludiAuthProvider } from './contexts/SludiAuthContext';
import type { AppConfig } from './types/config';
import './index.css';

if (!window.configs) {
  throw new Error(
    'Runtime configuration is missing: window.configs is not defined. ' +
    'Copy public/config.example.js to public/config.js and set the values.'
  );
}

const requiredConfigKeys: (keyof AppConfig)[] = [
  'consentEngineUrl',
  'idpBaseUrl',
  'idpClientId',
  'idpSignInRedirectUrl',
  'idpSignOutRedirectUrl',
];

const missingConfigKeys = requiredConfigKeys.filter((key) => !window.configs[key]);
if (missingConfigKeys.length > 0) {
  throw new Error(`Missing required runtime configuration: ${missingConfigKeys.join(', ')}`);
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <SludiAuthProvider>
      <BrowserRouter>
        <ConsentProvider>
          <App />
        </ConsentProvider>
      </BrowserRouter>
    </SludiAuthProvider>
  </StrictMode>,
);
