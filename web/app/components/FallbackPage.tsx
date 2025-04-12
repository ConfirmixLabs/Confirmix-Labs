'use client';

import { useState } from 'react';

export default function FallbackPage() {
  const [isChecking, setIsChecking] = useState(false);

  const handleRetry = () => {
    setIsChecking(true);
    // Sayfayı yeniden yükle
    window.location.reload();
  };

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-gray-50 px-4">
      <div className="max-w-lg w-full bg-white shadow-xl rounded-lg p-8 text-center">
        <div className="mb-6">
          <svg className="w-16 h-16 text-red-500 mx-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
        
        <h1 className="text-2xl font-bold text-gray-900 mb-3">Blockchain Server Connection Error</h1>
        
        <p className="text-gray-600 mb-6">
          Cannot connect to the blockchain backend server. Possible causes:
        </p>
        
        <div className="text-left mb-8 bg-gray-50 p-4 rounded-lg">
          <ul className="list-disc pl-5 space-y-2 text-sm text-gray-700">
            <li>The blockchain server (Go application) may not be running</li>
            <li>Access to the backend API may be blocked</li>
            <li>The backend API may be running at a different address</li>
            <li>The API endpoints of the backend API may have changed</li>
          </ul>
        </div>
        
        <div className="space-y-4">
          <h3 className="font-medium text-lg text-gray-800">To Solve This Problem:</h3>
          
          <div className="bg-blue-50 text-blue-800 p-4 rounded-lg text-left text-sm">
            <ol className="list-decimal pl-5 space-y-2">
              <li>Make sure the Go blockchain application is running</li>
              <li>Run the command <code className="bg-gray-200 px-1 py-0.5 rounded">go run main.go</code> in the terminal</li>
              <li>Check that the backend API address is <code className="bg-gray-200 px-1 py-0.5 rounded">http://localhost:8080/api</code></li>
              <li>Check API access permissions (CORS settings)</li>
            </ol>
          </div>
        </div>
        
        <div className="mt-8">
          <button
            onClick={handleRetry}
            disabled={isChecking}
            className={`px-6 py-3 rounded-md font-medium text-white ${
              isChecking ? 'bg-blue-400' : 'bg-blue-600 hover:bg-blue-700'
            } transition-colors duration-200 flex items-center justify-center w-full`}
          >
            {isChecking ? (
              <>
                <svg className="animate-spin -ml-1 mr-2 h-4 w-4 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                Checking...
              </>
            ) : (
              'Try Again'
            )}
          </button>
        </div>
      </div>
    </div>
  );
} 