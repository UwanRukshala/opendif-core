import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react';

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

function buildProfile(accessToken: string, idToken?: string): SludiUserProfile {
  if (idToken) {
    return decodeJwtPayload(idToken);
  }
  if (accessToken.includes('.')) {
    return decodeJwtPayload(accessToken);
  }
  return { sub: accessToken };
}

function parseHashParams(): URLSearchParams {
  const hash = window.location.hash.startsWith('#')
    ? window.location.hash.slice(1)
    : window.location.hash;
  return new URLSearchParams(hash);
}

const PENDING_AUTH_SESSION_KEY = 'sludi_pending_auth_session';

function clearAuthQueryParams() {
  const params = new URLSearchParams(window.location.search);
  params.delete('auth_session');
  params.delete('error');
  params.delete('error_description');
  const query = params.toString();
  const nextUrl = query
    ? `${window.location.pathname}?${query}`
    : window.location.pathname;
  window.history.replaceState({}, document.title, nextUrl);
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
      profile: buildProfile(token, idToken),
    };
  });
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const sessionFetchRef = useRef<string | null>(null);

  const persistUser = useCallback((nextUser: SludiUser) => {
    sessionStorage.setItem('sludi_access_token', nextUser.access_token);
    if (nextUser.id_token) {
      sessionStorage.setItem('sludi_id_token', nextUser.id_token);
    } else {
      sessionStorage.removeItem('sludi_id_token');
    }
    setUser(nextUser);
  }, []);

  const clearSession = useCallback(() => {
    sessionStorage.removeItem('sludi_access_token');
    sessionStorage.removeItem('sludi_id_token');
    setUser(null);
  }, []);

  const applyTokens = useCallback((tokens: {
    access_token: string;
    id_token?: string;
    token_type?: string;
    expires_in?: number;
  }) => {
    if (!tokens?.access_token) {
      throw new Error('Token response did not include an access token.');
    }

    const idToken = tokens.id_token;
    persistUser({
      access_token: tokens.access_token,
      id_token: idToken,
      profile: buildProfile(tokens.access_token, idToken),
    });
    setError(null);
  }, [persistUser]);

  useEffect(() => {
    const completeSignIn = async () => {
      const params = new URLSearchParams(window.location.search);
      const oauthError = params.get('error');
      const oauthErrorDescription = params.get('error_description');

      if (oauthError) {
        setError(oauthErrorDescription ?? oauthError);
        setIsLoading(false);
        clearAuthQueryParams();
        return;
      }

      const authSessionFromUrl = params.get('auth_session');
      if (authSessionFromUrl) {
        sessionStorage.setItem(PENDING_AUTH_SESSION_KEY, authSessionFromUrl);
        clearAuthQueryParams();
      }
      const authSession =
        authSessionFromUrl ?? sessionStorage.getItem(PENDING_AUTH_SESSION_KEY);

      if (authSession) {
        const handledKey = `sludi_session_handled_${authSession}`;
        if (sessionStorage.getItem(handledKey) === '1') {
          sessionStorage.removeItem(PENDING_AUTH_SESSION_KEY);
          setIsLoading(false);
          return;
        }
        if (sessionFetchRef.current === authSession) {
          return;
        }
        sessionFetchRef.current = authSession;

        try {
          setIsLoading(true);
          setError(null);

          const response = await fetch(
            `${window.configs.consentEngineUrl}/auth/session?session=${encodeURIComponent(authSession)}`,
          );

          if (!response.ok) {
            let message = `Token fetch failed (HTTP ${response.status})`;
            try {
              const body = await response.json();
              if (body?.message) message = body.message;
            } catch {
              // ignore parse errors
            }
            throw new Error(message);
          }

          const tokens = await response.json();
          applyTokens(tokens);
          sessionStorage.setItem(handledKey, '1');
          sessionStorage.removeItem(PENDING_AUTH_SESSION_KEY);
        } catch (err) {
          const message = err instanceof Error ? err.message : 'Sign in failed';
          setError(message);
          if (!sessionStorage.getItem('sludi_access_token')) {
            clearSession();
          }
        } finally {
          sessionFetchRef.current = null;
          setIsLoading(false);
        }
        return;
      }

      const hashParams = parseHashParams();
      const accessToken = hashParams.get('access_token');
      const idToken = hashParams.get('id_token') ?? undefined;

      if (accessToken) {
        applyTokens({ access_token: accessToken, id_token: idToken });
        window.history.replaceState({}, document.title, window.location.pathname + window.location.search);
      }

      setIsLoading(false);
    };

    void completeSignIn();
  }, [applyTokens, clearSession]);

  const signIn = useCallback(async () => {
    const configs = window.configs;
    setError(null);

    const returnTo = `${window.location.origin}${window.location.pathname}${window.location.search}`;
    const loginUrl = new URL(`${configs.consentEngineUrl}/auth/login`);
    loginUrl.searchParams.set('return_to', returnTo);

    window.location.assign(loginUrl.toString());
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
