'use client';

import { useState, useCallback, useMemo } from 'react';
import { useBlockchain } from './contexts/BlockchainContext';
import ErrorMessage from './components/ErrorMessage';
import FallbackPage from './components/FallbackPage';
import WalletPanel from './components/WalletPanel';
import ValidatorPanel from './components/ValidatorPanel';
import BlocksTable from './components/BlocksTable';
import TransactionsTable from './components/TransactionsTable';
import Header from './components/Header';

export default function Home() {
  // Use blockchain context
  const {
    status,
    blocks,
    transactions,
    pendingTransactions,
    validators,
    loading,
    refreshing,
    error,
    connectionError,
    usingMockData,
    wallet,
    fetchData,
    setWallet
  } = useBlockchain();
  
  // Local state only for UI
  const [activePanel, setActivePanel] = useState<'overview' | 'wallet' | 'validator' | 'blocks' | 'transactions'>('overview');

  const handlePanelChange = useCallback((panel: 'overview' | 'wallet' | 'validator' | 'blocks' | 'transactions') => {
    setActivePanel(panel);
  }, []);

  // Memoize sliced arrays for overview panel
  const recentBlocks = useMemo(() => blocks.slice(0, 5), [blocks]);
  const recentTransactions = useMemo(() => transactions.slice(0, 5), [transactions]);

  // Show FallbackPage if cannot connect to the API server
  if (connectionError) {
    return <FallbackPage />;
  }

  if (loading && !status) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p>Loading blockchain data...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <ErrorMessage 
        message={error} 
        action={() => window.location.reload()} 
        actionText="Refresh Page" 
      />
    );
  }

  if (!status) {
    return (
      <ErrorMessage 
        message="Cannot access blockchain data. Please try again later." 
        action={() => window.location.reload()} 
        actionText="Try Again" 
      />
    );
  }

  return (
    <main className="container mx-auto px-4 py-8">
      {usingMockData && (
        <div className="bg-yellow-100 text-yellow-800 px-4 py-2 text-sm text-center mb-4 rounded">
          There is a problem accessing the API server. The data shown may not be up to date.
          <button 
            onClick={() => window.location.reload()} 
            className="ml-2 underline hover:no-underline"
          >
            Refresh
          </button>
        </div>
      )}
      
      {/* Header Component */}
      <Header 
        status={status}
        transactionCount={pendingTransactions.length}
        validatorCount={validators.length}
        usingMockData={usingMockData}
        onRefresh={fetchData}
      />
      
      {/* Main Menu */}
      <div className="bg-white rounded-lg shadow-md p-6 mb-6">
        <div className="flex flex-wrap gap-2 justify-center">
          <button
            onClick={() => handlePanelChange('overview')}
            className={`px-4 py-3 rounded-md text-sm font-medium flex items-center ${
              activePanel === 'overview'
                ? 'bg-blue-500 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            <svg className="w-5 h-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
            </svg>
            Overview
          </button>
          <button
            onClick={() => handlePanelChange('wallet')}
            className={`px-4 py-3 rounded-md text-sm font-medium flex items-center ${
              activePanel === 'wallet'
                ? 'bg-blue-500 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            <svg className="w-5 h-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
            </svg>
            Wallet Operations
          </button>
          <button
            onClick={() => handlePanelChange('validator')}
            className={`px-4 py-3 rounded-md text-sm font-medium flex items-center ${
              activePanel === 'validator'
                ? 'bg-blue-500 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            <svg className="w-5 h-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
            </svg>
            Validator Operations
          </button>
          <button
            onClick={() => handlePanelChange('blocks')}
            className={`px-4 py-3 rounded-md text-sm font-medium flex items-center ${
              activePanel === 'blocks'
                ? 'bg-blue-500 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            <svg className="w-5 h-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
            </svg>
            All Blocks
          </button>
          <button
            onClick={() => handlePanelChange('transactions')}
            className={`px-4 py-3 rounded-md text-sm font-medium flex items-center ${
              activePanel === 'transactions'
                ? 'bg-blue-500 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            <svg className="w-5 h-5 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
            </svg>
            Transactions
          </button>
        </div>
      </div>
      
      {/* Active Panel Content */}
      {activePanel === 'overview' && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="bg-white rounded-lg shadow-md p-6">
            <h2 className="text-xl font-bold mb-4">Recent Blocks</h2>
            <BlocksTable blocks={recentBlocks} loading={refreshing} />
            <div className="mt-4 text-right">
              <button 
                onClick={() => handlePanelChange('blocks')} 
                className="text-blue-600 hover:text-blue-800"
              >
                View all blocks →
              </button>
            </div>
          </div>
          
          <div className="bg-white rounded-lg shadow-md p-6">
            <h2 className="text-xl font-bold mb-4">Recent Transactions</h2>
            <TransactionsTable transactions={recentTransactions} loading={refreshing} />
            <div className="mt-4 text-right">
              <button 
                onClick={() => handlePanelChange('transactions')} 
                className="text-blue-600 hover:text-blue-800"
              >
                View all transactions →
              </button>
            </div>
          </div>
        </div>
      )}
      
      {activePanel === 'wallet' && (
        <WalletPanel 
          wallet={wallet}
          onRefresh={fetchData}
        />
      )}
      
      {activePanel === 'validator' && (
        <ValidatorPanel 
          wallet={wallet}
          validators={validators}
          onRefresh={fetchData}
        />
      )}
      
      {activePanel === 'blocks' && (
        <div className="bg-white rounded-lg shadow-md p-6">
          <h2 className="text-xl font-bold mb-4">All Blocks</h2>
          <BlocksTable blocks={blocks} loading={loading} />
        </div>
      )}
      
      {activePanel === 'transactions' && (
        <div className="bg-white rounded-lg shadow-md p-6">
          <h2 className="text-xl font-bold mb-4">All Transactions</h2>
          <TransactionsTable transactions={transactions} loading={loading} />
        </div>
      )}
    </main>
  );
}
