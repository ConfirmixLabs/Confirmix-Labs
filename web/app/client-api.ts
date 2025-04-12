import { BlockchainStatus, Block, Transaction, ValidatorInfo } from './lib/types';

export const api = {
  getStatus: async (): Promise<BlockchainStatus> => {
    const response = await fetch('/api/status');
    if (!response.ok) {
      throw new Error('API connection error');
    }
    const data = await response.json();
    if (data.error) {
      throw new Error(data.error);
    }
    return data;
  },

  getBlocks: async (): Promise<Block[]> => {
    const response = await fetch('/api/blocks');
    if (!response.ok) {
      throw new Error('API connection error');
    }
    const data = await response.json();
    if (data.error) {
      throw new Error(data.error);
    }
    return data;
  },

  getTransactions: async (): Promise<Transaction[]> => {
    const response = await fetch('/api/transactions');
    if (!response.ok) {
      throw new Error('API connection error');
    }
    const data = await response.json();
    if (data.error) {
      throw new Error(data.error);
    }
    return data;
  },

  getPendingTransactions: async (): Promise<Transaction[]> => {
    const response = await fetch('/api/transactions/pending');
    if (!response.ok) {
      throw new Error('API connection error');
    }
    const data = await response.json();
    if (data.error) {
      throw new Error(data.error);
    }
    return data;
  },

  getConfirmedTransactions: async (): Promise<Transaction[]> => {
    const response = await fetch('/api/transactions/confirmed');
    if (!response.ok) {
      throw new Error('API connection error');
    }
    const data = await response.json();
    if (data.error) {
      throw new Error(data.error);
    }
    return data;
  },

  getValidators: async (): Promise<ValidatorInfo[]> => {
    const response = await fetch('/api/validators');
    if (!response.ok) {
      throw new Error('API connection error');
    }
    const data = await response.json();
    if (data.error) {
      throw new Error(data.error);
    }
    return data;
  },

  registerValidator: async (address: string, humanProof: string): Promise<ValidatorInfo> => {
    const response = await fetch('/api/validator/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ address, humanProof })
    });
    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || 'API connection error');
    }
    const data = await response.json();
    if (data.error) {
      throw new Error(data.error);
    }
    return data;
  },

  createWallet: async (): Promise<{ address: string; publicKey: string; privateKey: string }> => {
    const response = await fetch('/api/wallet/create', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' }
    });
    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || 'API connection error');
    }
    const data = await response.json();
    if (data.error) {
      throw new Error(data.error);
    }
    return data;
  },

  importWallet: async (privateKey: string): Promise<{ address: string; publicKey: string; privateKey: string; exists: boolean }> => {
    const response = await fetch('/api/wallet/import', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ privateKey })
    });
    if (!response.ok) {
      const errorData = await response.json();
      throw new Error(errorData.error || 'API connection error');
    }
    const data = await response.json();
    if (data.error) {
      throw new Error(data.error);
    }
    return data;
  },

  async getBalance(address: string): Promise<string> {
    try {
      console.log(`Requesting balance for address: ${address}`);
      const response = await fetch(`/api/wallet/balance/${address}`);
      if (!response.ok) {
        throw new Error('Could not retrieve balance information');
      }
      
      const data = await response.json();
      console.log("Raw API balance response:", data);
      
      if (!data || data.balance === null || data.balance === undefined) {
        console.log("API returned null/undefined balance, returning '0'");
        return "0";
      }
      
      if (typeof data.balance !== 'string') {
        console.log(`API returned non-string balance: ${typeof data.balance}, value: ${data.balance}`);
        // Try to convert to string
        return String(data.balance);
      }
      
      return data.balance;
    } catch (error) {
      console.error('Balance query error:', error);
      throw new Error('Could not retrieve balance information');
    }
  },

  transfer: async (from: string, to: string, value: string): Promise<any> => {
    const response = await fetch('/api/transaction', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ from, to, value })
    });
    
    const data = await response.json();
    if (!response.ok || data.error) {
      throw new Error(data.error || 'Transfer transaction failed');
    }
    return data;
  }
}; 