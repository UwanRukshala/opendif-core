import { Shield } from 'lucide-react';
import React, { useEffect } from 'react';
import { useNavigate } from "react-router-dom";
import SludiSignInButton from '../components/SludiSignInButton';
import { useSludiAuth } from '../contexts/SludiAuthContext';

const LoginPage: React.FC = () => {
  const navigate = useNavigate();
  const { isAuthenticated, isLoading, signIn, error } = useSludiAuth();

  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      const consentId = localStorage.getItem('consentId');
      if (consentId) {
        navigate('/');
      } else {
        navigate('/error');
      }
    }
  }, [isAuthenticated, isLoading, navigate]);

  if (isLoading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600 mx-auto mb-4"></div>
          <p className="text-gray-600">Loading...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 flex items-center justify-center p-4 relative">
      <div className="max-w-md w-full bg-white rounded-lg shadow-lg p-6 text-center">
        <Shield className="h-12 w-12 text-blue-500 mx-auto mb-4" />
        <h1 className="text-2xl font-bold text-gray-800 mb-2">Consent Portal</h1>
        <p className="text-gray-600 mb-6">
          Sign in with SLUDI to review and approve your consent request.
        </p>
        {error && (
          <p className="mb-4 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700">{error}</p>
        )}
        <SludiSignInButton onClick={() => void signIn()} />
      </div>
    </div>
  );
};

export default LoginPage;
