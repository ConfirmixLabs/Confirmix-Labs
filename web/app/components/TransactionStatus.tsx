import React from 'react';

type TransactionStatusProps = {
  error?: string;
  message?: string;
  details?: string;
  isLoading?: boolean;
};

export default function TransactionStatus({ error, message, details, isLoading }: TransactionStatusProps) {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-4 bg-blue-50 text-blue-700 rounded-lg">
        <svg className="animate-spin h-5 w-5 mr-3" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
        <span>İşlem gönderiliyor...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4 bg-red-50 text-red-700 rounded-lg">
        <div className="font-semibold mb-1">{error}</div>
        {message && <div className="text-sm mb-1">{message}</div>}
        {details && <div className="text-xs text-red-600">{details}</div>}
      </div>
    );
  }

  if (message) {
    return (
      <div className="p-4 bg-green-50 text-green-700 rounded-lg">
        <div className="font-semibold">{message}</div>
      </div>
    );
  }

  return null;
} 