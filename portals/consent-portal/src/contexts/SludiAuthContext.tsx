import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react';
import { generateCodeChallenge, generateRandomString } from '../utils/pkce';

const PKCE_VERIFIER_KEY = 'sludi_pkce_verifier';
const OIDC_STATE_KEY = 'sludi_oidc_state';

interface SludiUserProfile {
  sub?: string;
  email?: string;
  name?: string;
  given_name?: string;
  preferred_username?: string;
}

export interface SludiUser {
  access_token: string;
  id_token?: string;
  profile: SludiUserProfile;
}

interface SludiAuthContextType {
  user: SludiUser | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
  signIn: () => Promise<void>;
  signOut: () => void;
}

const SludiAuthContext = createContext<SludiAuthContextType | undefined>(undefined);

function decodeJwtPayload(token: string): SludiUserProfile {
  try {
    const payload = token.split('.')[1];
    if (!payload) return {};
    const json = atob(payload.replace(/-/g, '+').replace(/_/g, '/'));
    return JSON.parse(json) as SludiUserProfile;
  } catch {
    return {};
  }
}

function buildAuthorizeUrl(
  baseUrl: string,
  params: Record<string, string>,
): string {
  const url = new URL('/authorize', baseUrl);
  Object.entries(params).forEach(([key, value]) => {
    url.searchParams.set(key, value);
  });
  return url.toString();
}

// eslint-disable-next-line react-refresh/only-export-components
export const useSludiAuth = () => {
  const context = useContext(SludiAuthContext);
  if (!context) {
    throw new Error('useSludiAuth must be used within a SludiAuthProvider');
  }
  return context;
};

export const SludiAuthProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<SludiUser | null>(() => {
    const token = sessionStorage.getItem('sludi_access_token');
    const idToken = sessionStorage.getItem('sludi_id_token') ?? undefined;
    if (!token) return null;
    return {
      access_token: token,
      id_token: idToken,
      profile: idToken ? decodeJwtPayload(idToken) : {},
    };
  });
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const persistUser = useCallback((nextUser: SludiUser) => {
    sessionStorage.setItem('sludi_access_token', nextUser.access_token);
    if (nextUser.id_token) {
      sessionStorage.setItem('sludi_id_token', nextUser.id_token);
    }
    setUser(nextUser);
  }, []);

  const clearSession = useCallback(() => {
    sessionStorage.removeItem('sludi_access_token');
    sessionStorage.removeItem('sludi_id_token');
    sessionStorage.removeItem(PKCE_VERIFIER_KEY);
    sessionStorage.removeItem(OIDC_STATE_KEY);
    setUser(null);
  }, []);

  const exchangeAuthorizationCode = useCallback(async (code: string) => {
    const configs = window.configs;
    const codeVerifier = sessionStorage.getItem(PKCE_VERIFIER_KEY);
    if (!codeVerifier) {
      throw new Error('Missing PKCE verifier. Please try signing in again.');
    }

    const response = await fetch(`${configs.consentEngineUrl}/auth/token`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        code,
        redirect_uri: configs.idpSignInRedirectUrl,
        code_verifier: codeVerifier,
      }),
    });

    if (!response.ok) {
      let message = `Token exchange failed (HTTP ${response.status})`;
      try {
        const body = await response.json();
        if (body?.message) message = body.message;
      } catch {
        // ignore parse errors
      }
      throw new Error(message);
    }

    const tokens = await response.json();
    if (!tokens?.access_token) {
      throw new Error('Token exchange did not return an access token.');
    }

    const nextUser: SludiUser = {
      access_token: tokens.access_token,
      id_token: tokens.id_token,
      profile: tokens.id_token ? decodeJwtPayload(tokens.id_token) : {},
    };

    persistUser(nextUser);
    sessionStorage.removeItem(PKCE_VERIFIER_KEY);
    sessionStorage.removeItem(OIDC_STATE_KEY);
  }, [persistUser]);

  useEffect(() => {
    const completeSignIn = async () => {
      const params = new URLSearchParams(window.location.search);
      const code = params.get('code');
      const state = params.get('state');
      const oauthError = params.get('error');

      if (oauthError) {
        setError(params.get('error_description') ?? oauthError);
        setIsLoading(false);
        window.history.replaceState({}, document.title, window.location.pathname);
        return;
      }

      if (!code) {
        setIsLoading(false);
        return;
      }

      const expectedState = sessionStorage.getItem(OIDC_STATE_KEY);
      if (!expectedState || state !== expectedState) {
        setError('Invalid login state. Please try signing in again.');
        setIsLoading(false);
        window.history.replaceState({}, document.title, window.location.pathname);
        return;
      }

      try {
        setIsLoading(true);
        setError(null);
        await exchangeAuthorizationCode(code);
        window.history.replaceState({}, document.title, window.location.pathname);
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Sign in failed';
        setError(message);
        clearSession();
        window.history.replaceState({}, document.title, window.location.pathname);
      } finally {
        setIsLoading(false);
      }
    };

    void completeSignIn();
  }, [clearSession, exchangeAuthorizationCode]);

  const signIn = useCallback(async () => {
    const configs = window.configs;
    setError(null);

    const state = generateRandomString(16);
    const nonce = generateRandomString(16);
    const codeVerifier = generateRandomString(32);
    const codeChallenge = await generateCodeChallenge(codeVerifier);

    sessionStorage.setItem(PKCE_VERIFIER_KEY, codeVerifier);
    sessionStorage.setItem(OIDC_STATE_KEY, state);

    const authorizeUrl = buildAuthorizeUrl(configs.idpBaseUrl, {
      client_id: configs.idpClientId,
      redirect_uri: configs.idpSignInRedirectUrl,
      response_type: 'code',
      scope: configs.idpScope || 'openid profile email',
      state,
      nonce,
      code_challenge: codeChallenge,
      code_challenge_method: 'S256',
    });

    window.location.assign(authorizeUrl);
  }, []);

  const signOut = useCallback(() => {
    clearSession();
    setError(null);
    const redirectUri = window.configs.idpSignOutRedirectUrl;
    window.location.assign(redirectUri);
  }, [clearSession]);

  const value = useMemo(
    () => ({
      user,
      isAuthenticated: user !== null,
      isLoading,
      error,
      signIn,
      signOut,
    }),
    [user, isLoading, error, signIn, signOut],
  );

  return <SludiAuthContext.Provider value={value}>{children}</SludiAuthContext.Provider>;
};
