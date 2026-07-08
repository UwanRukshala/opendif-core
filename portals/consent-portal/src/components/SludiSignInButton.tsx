import React from 'react';
import { LogIn } from 'lucide-react';

interface SludiSignInButtonProps {
  onClick: () => void;
  disabled?: boolean;
  className?: string;
}

const SludiSignInButton: React.FC<SludiSignInButtonProps> = ({
  onClick,
  disabled = false,
  className = '',
}) => {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={`flex w-full items-center justify-center gap-3 rounded-lg border-2 border-[#1e3a5f] bg-[#1e3a5f] px-6 py-3 text-base font-semibold text-white transition-colors hover:bg-[#152a45] disabled:cursor-not-allowed disabled:opacity-60 ${className}`}
    >
      <LogIn className="h-5 w-5" aria-hidden="true" />
      Sign in with SLUDI
    </button>
  );
};

export default SludiSignInButton;
