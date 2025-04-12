'use client';

import React, { createContext, useContext, useCallback, useEffect, useState, useMemo } from 'react';
import { api } from '../client-api';
import { BlockchainStatus, Block, Transaction } from '../lib/types';
import { log } from 'console';

interface BlockchainContextType {
  status: BlockchainStatus | null;
  blocks: Block[];
  transactions: Transaction[];
  pendingTransactions: Transaction[];
  validators: Array<{ address: string; humanProof: string }>;
  loading: boolean;
  refreshing: boolean;
  error: string | null;
  connectionError: boolean;
  usingMockData: boolean;
  wallet: { address: string; publicKey: string; privateKey?: string } | null;
  fetchData: () => Promise<void>;
  setWallet: (wallet: { address: string; publicKey: string; privateKey?: string } | null) => void;
  createWallet: () => Promise<{ address: string; publicKey: string; privateKey: string }>;
  importWallet: (privateKey: string) => Promise<{ address: string; publicKey: string; privateKey: string }>;
  checkBalance: (address: string) => Promise<string>;
  transfer: (to: string, amount: string) => Promise<{ message?: string; warning?: string }>;
  registerValidator: (humanProof: string) => Promise<void>;
}

const BlockchainContext = createContext<BlockchainContextType | undefined>(undefined);

export function BlockchainProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<BlockchainStatus | null>(null);
  const [blocks, setBlocks] = useState<Block[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [pendingTransactions, setPendingTransactions] = useState<Transaction[]>([]);
  const [validators, setValidators] = useState<Array<{ address: string; humanProof: string }>>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [connectionError, setConnectionError] = useState(false);
  const [usingMockData, setUsingMockData] = useState(false);
  const [wallet, setWallet] = useState<{ address: string; publicKey: string; privateKey?: string } | null>(null);
  const isInitialLoadRef = React.useRef(true);
  const loadingRef = React.useRef(true);
  const pollingIntervalRef = React.useRef<NodeJS.Timeout | null>(null); 
  const shouldPollRef = React.useRef(true);

  // Update ref whenever loading state changes
  useEffect(() => {
    loadingRef.current = loading;
  }, [loading]);

  const fetchData = useCallback(async (isPolling = false) => {
    // Only update loading/refreshing state if this is not a silent background poll
    if (!isPolling) {
      if (isInitialLoadRef.current) {
        setLoading(true);
        isInitialLoadRef.current = false;
      } else {
        setRefreshing(true);
      }
    }
    
    setError(null);
    
    try {
      const [newStatus, newBlocks, newTransactions, newPendingTransactions, newValidators] = await Promise.all([
        api.getStatus(),
        api.getBlocks(),
        api.getTransactions(),
        api.getPendingTransactions(),
        api.getValidators()
      ]);

      // Simple length check to detect changes (not perfect but sufficient for basic change detection)
      const blocksChanged = newBlocks.length !== blocks.length;
      const txChanged = newTransactions.length !== transactions.length;
      const pendingTxChanged = newPendingTransactions.length !== pendingTransactions.length;
      const validatorsChanged = newValidators.length !== validators.length;
      
      const hasChanged = blocksChanged || txChanged || pendingTxChanged || validatorsChanged || !status;
      
      if (hasChanged) {
        setStatus(newStatus);
        setBlocks(newBlocks);
        setTransactions(newTransactions);
        setPendingTransactions(newPendingTransactions);
        setValidators(newValidators);
        setConnectionError(false);
        setUsingMockData(false);
      }
      
    } catch (error) {
      console.error('Data loading error:', error);
      setError('Could not connect to backend server. Please make sure the blockchain server is running.');
      setConnectionError(true);
      setUsingMockData(false);
    } finally {
      // Only update loading/refreshing state if this is not a silent background poll
      if (!isPolling) {
        setLoading(false);
        setRefreshing(false);
      }
    }
  }, [status, blocks, transactions, pendingTransactions, validators]);

  // Function to pause polling (e.g. when tab is in background)
  const pausePolling = useCallback(() => {
    shouldPollRef.current = false;
  }, []);

  // Function to resume polling
  const resumePolling = useCallback(() => {
    shouldPollRef.current = true;
  }, []);

  // Setup polling on mount
  useEffect(() => {
    // Initial fetch (non-polling)
    fetchData(false);
    
    // Setup interval for polling with shouldPoll check
    const setupPolling = () => {
      pollingIntervalRef.current = setInterval(() => {
        if (shouldPollRef.current) {
          fetchData(true); // Pass true to indicate this is a polling call
        }
      }, 5000);
    };
    
    setupPolling();
    
    // Add visibility change listener to pause/resume polling when tab is hidden/visible
    document.addEventListener('visibilitychange', () => {
      if (document.hidden) {
        pausePolling();
      } else {
        resumePolling();
      }
    });
    
    return () => {
      if (pollingIntervalRef.current) {
        clearInterval(pollingIntervalRef.current);
      }
    };
  }, [fetchData, pausePolling, resumePolling]);

  // New wallet operations
  const createWallet = useCallback(async () => {
    try {
      const newWallet = await api.createWallet();
      setWallet(newWallet);
      return newWallet;
    } catch (error) {
      throw error;
    }
  }, []);

  const importWallet = useCallback(async (privateKey: string) => {
    try {
      const newWallet = await api.importWallet(privateKey);
      setWallet(newWallet);
      return newWallet;
    } catch (error) {
      throw error;
    }
  }, []);

  const checkBalance = useCallback(async (address: string) => {
    try {
      const balance = await api.getBalance(address);
      return balance;
    } catch (error) {
      throw error;
    }
  }, []);

  const transfer = useCallback(async (to: string, amount: string) => {
    if (!wallet) throw new Error('No wallet available');
    
    try {
      const response = await fetch('/api/transaction', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          from: wallet.address,
          to,
          value: amount
        }),
      });

      const data = await response.json();
      if (data.error) throw new Error(data.error);
      
      // Refresh data after successful transfer
      await fetchData();
      
      return data;
    } catch (error) {
      throw error;
    }
  }, [wallet, fetchData]);

  // New validator operations
  const registerValidator = useCallback(async (humanProof: string) => {
    if (!wallet) throw new Error('No wallet available');
    
    try {
      await api.registerValidator(wallet.address, humanProof);
      await fetchData(); // Refresh data after registration
    } catch (error) {
      throw error;
    }
  }, [wallet, fetchData]);

  const value = useMemo(() => ({
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
    setWallet,
    createWallet,
    importWallet,
    checkBalance,
    transfer,
    registerValidator,
  }), [
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
    createWallet,
    importWallet,
    checkBalance,
    transfer,
    registerValidator
  ]);

  return (
    <BlockchainContext.Provider value={value}>
      {children}
    </BlockchainContext.Provider>
  );
}

export function useBlockchain() {
  const context = useContext(BlockchainContext);
  if (context === undefined) {
    throw new Error('useBlockchain must be used within a BlockchainProvider');
  }
  return context;
} 