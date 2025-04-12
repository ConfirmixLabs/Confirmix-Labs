'use client';

import { useEffect } from 'react';

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error('Page error:', error);
  }, [error]);

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-100">
      <div className="bg-white p-8 rounded-lg shadow-md max-w-md w-full">
        <div className="flex items-center justify-center w-12 h-12 mx-auto mb-4 rounded-full bg-red-100">
          <svg className="w-6 h-6 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path>
          </svg>
        </div>
        <h2 className="mb-4 text-xl font-semibold text-center text-gray-800">An Error Occurred</h2>
        <div className="mb-4 p-3 bg-red-50 rounded-md text-sm font-mono text-red-800 overflow-auto max-h-40">
          {error.message || 'Beklenmeyen bir hata oluştu'}
        </div>
        <p className="mb-6 text-center text-gray-600">
          Cannot connect to the API server. Please make sure the server is running.
        </p>
        <div className="flex space-x-4">
          <button
            onClick={() => reset()}
            className="flex-1 py-2 px-4 bg-blue-600 hover:bg-blue-700 text-white font-semibold rounded-md transition-colors"
          >
            Try Again
          </button>
          <button
            onClick={() => window.location.reload()}
            className="flex-1 py-2 px-4 bg-gray-200 hover:bg-gray-300 text-gray-800 font-semibold rounded-md transition-colors"
          >
            Refresh Page
          </button>
        </div>
      </div>
    </div>
  );
} 