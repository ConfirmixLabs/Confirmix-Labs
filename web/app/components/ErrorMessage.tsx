'use client';

import { useState } from 'react';

type ErrorMessageProps = {
  message: string;
  action?: () => void;
  actionText?: string;
  title?: string;
  suggestion?: string;
};

export default function ErrorMessage({
  message,
  action,
  actionText = 'Try Again',
  title = 'An Error Occurred',
  suggestion
}: ErrorMessageProps) {
  const [isPerformingAction, setIsPerformingAction] = useState(false);

  const handleAction = () => {
    if (action) {
      setIsPerformingAction(true);
      action();
    }
  };

  // Hata mesajının türüne göre öneriler oluştur
  const getSuggestions = () => {
    if (suggestion) return suggestion;
    
    if (message.includes('Backend connection error') || message.includes('connection')) {
      return 'Make sure the blockchain server (Go application) is running. Start it using the "go run main.go" command in the terminal.';
    }
    
    if (message.includes('timeout') || message.includes('timed out')) {
      return 'The blockchain server is taking too long to respond. Make sure the server is not overloaded.';
    }
    
    if (message.includes('Failed to fetch') || message.includes('could not load')) {
      return 'Cannot access API endpoints. Make sure the backend server\'s API structure is in the expected format.';
    }
    
    return 'Try refreshing the page. If the problem persists, check the backend server.';
  };

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-gray-50 px-4">
      <div className="max-w-lg w-full bg-white shadow-xl rounded-lg p-8">
        <div className="mb-6 flex items-center">
          <div className="bg-red-100 p-3 rounded-full mr-4">
            <svg className="w-6 h-6 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>
          <h2 className="text-xl font-bold text-gray-900">{title}</h2>
        </div>
        
        <div className="mb-6">
          <div className="bg-red-50 border border-red-100 rounded-md p-4 mb-4">
            <p className="text-red-800 font-medium">Error Message:</p>
            <p className="text-red-700 mt-1">{message}</p>
          </div>
          
          <div className="bg-blue-50 border border-blue-100 rounded-md p-4">
            <p className="text-blue-800 font-medium">Suggestion:</p>
            <p className="text-blue-700 mt-1">{getSuggestions()}</p>
          </div>
        </div>
        
        {action && (
          <button
            onClick={handleAction}
            disabled={isPerformingAction}
            className={`w-full px-6 py-3 rounded-md font-medium text-white ${
              isPerformingAction ? 'bg-blue-400' : 'bg-blue-600 hover:bg-blue-700'
            } transition-colors duration-200 flex items-center justify-center`}
          >
            {isPerformingAction ? (
              <>
                <svg className="animate-spin -ml-1 mr-2 h-4 w-4 text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                Processing...
              </>
            ) : (
              actionText
            )}
          </button>
        )}
      </div>
    </div>
  );
} 