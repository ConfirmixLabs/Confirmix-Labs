'use client';

import { useState, useEffect } from 'react';
import { api } from '../client-api';
import ValidatorList from './ValidatorList';
import ValidatorForm from './ValidatorForm';

type ValidatorPanelProps = {
  wallet: { address: string; publicKey: string } | null;
  validators: Array<{ address: string; humanProof: string }>;
  onRefresh: () => void;
};

export default function ValidatorPanel({ wallet, validators, onRefresh }: ValidatorPanelProps) {
  const [activeTab, setActiveTab] = useState<'overview' | 'register'>('overview');
  const [isValidator, setIsValidator] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  useEffect(() => {
    if (wallet) {
      setIsValidator(validators.some(v => v.address === wallet.address));
    }
  }, [wallet, validators]);

  const handleRegister = async (humanProof: string) => {
    if (!wallet) {
      setError('Please create a wallet first');
      return;
    }

    try {
      setError(null);
      setSuccess(null);
      const result = await api.registerValidator(wallet.address, humanProof);
      setSuccess('Successfully registered as a validator');
      setIsValidator(true);
      onRefresh();
    } catch (err) {
      let errorMessage = err instanceof Error ? err.message : 'An error occurred during validator registration';

      // Special message when API endpoint is not found
      if (errorMessage.includes('HTTP 404') || errorMessage.includes('page not found')) {
        errorMessage = 'API endpoint for validator registration not found. You may not be using the latest version of the backend server. Please update your backend API code.';
      }

      setError(errorMessage);
    }
  };

  return (
    <div className="bg-white rounded-lg shadow">
      <div className="px-4 py-5 sm:p-6">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-2xl font-bold text-gray-900">Validator Panel</h2>
          <div className="flex space-x-2">
            <button
              onClick={() => setActiveTab('overview')}
              className={`px-4 py-2 rounded-md text-sm font-medium ${activeTab === 'overview'
                  ? 'bg-purple-100 text-purple-700'
                  : 'text-gray-500 hover:text-gray-700'
                }`}
            >
              Overview
            </button>
            {!isValidator && (
              <button
                onClick={() => setActiveTab('register')}
                className={`px-4 py-2 rounded-md text-sm font-medium ${activeTab === 'register'
                    ? 'bg-purple-100 text-purple-700'
                    : 'text-gray-500 hover:text-gray-700'
                  }`}
              >
                Become Validator
              </button>
            )}
          </div>
        </div>

        {error && (
          <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-md">
            <div className="flex">
              <div className="flex-shrink-0">
                <svg className="h-5 w-5 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
              <div className="ml-3">
                <p className="text-sm text-red-700">{error}</p>
              </div>
            </div>
          </div>
        )}

        {success && (
          <div className="mb-4 p-4 bg-green-50 border border-green-200 rounded-md">
            <div className="flex">
              <div className="flex-shrink-0">
                <svg className="h-5 w-5 text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
              <div className="ml-3">
                <p className="text-sm text-green-700">{success}</p>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'overview' && (
          <div>
            <h3 className="text-lg font-medium text-gray-900 mb-4">Aktif Validatorlar</h3>
            <ValidatorList validators={validators} currentAddress={wallet?.address} />
          </div>
        )}

        {activeTab === 'register' && (
          <ValidatorForm
            onSubmit={handleRegister}
            onCancel={() => setActiveTab('overview')}
            loading={false}
          />
        )}
      </div>
    </div>
  );
} 