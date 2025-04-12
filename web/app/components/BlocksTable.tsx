'use client';

import { useState } from 'react';
import { Block, Transaction } from '../lib/types';

type BlocksTableProps = {
  blocks?: Block[];
  loading?: boolean;
};

export default function BlocksTable({ blocks = [], loading }: BlocksTableProps) {
  const [selectedBlock, setSelectedBlock] = useState<Block | null>(null);

  const openBlockDetails = (block: Block) => {
    setSelectedBlock(block);
  };

  const closeBlockDetails = () => {
    setSelectedBlock(null);
  };

  if (loading) {
    return (
      <div className="bg-white rounded-lg shadow-md p-6 mb-6">
        <div className="flex items-center space-x-3 mb-4">
          <div className="w-8 h-8 bg-blue-100 rounded-md flex items-center justify-center">
            <svg className="w-5 h-5 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
            </svg>
          </div>
          <h2 className="text-xl font-bold text-gray-800">Recent Blocks</h2>
        </div>
        <div className="animate-pulse">
          <div className="h-10 bg-gray-200 rounded mb-4"></div>
          <div className="space-y-3">
            <div className="h-12 bg-gray-100 rounded"></div>
            <div className="h-12 bg-gray-100 rounded"></div>
            <div className="h-12 bg-gray-100 rounded"></div>
          </div>
        </div>
      </div>
    );
  }
  
  if (blocks.length === 0) {
    return (
      <div className="text-center py-8">
        <svg className="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
        </svg>
        <h2 className="mt-2 text-xl font-bold text-gray-800">Recent Blocks</h2>
        <p className="mt-1 text-gray-500">No blocks have been mined yet.</p>
      </div>
    );
  }
  
  return (
    <div className="bg-white rounded-lg shadow-md p-6 mb-6">
      <div className="flex items-center space-x-3 mb-4">
        <div className="w-8 h-8 bg-blue-100 rounded-md flex items-center justify-center">
          <svg className="w-5 h-5 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
          </svg>
        </div>
        <h2 className="text-xl font-bold text-gray-800">Recent Blocks</h2>
      </div>
      
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider rounded-tl-lg">Index</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Hash</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Time</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Validator</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider rounded-tr-lg">Transactions</th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {blocks.map((block) => (
              <tr 
                key={block.Hash} 
                className="hover:bg-gray-50 cursor-pointer" 
                onClick={() => openBlockDetails(block)}
              >
                <td className="px-4 py-3 whitespace-nowrap">
                  <span className="inline-flex items-center justify-center w-8 h-8 bg-blue-100 text-blue-800 text-sm font-medium rounded-full">
                    {block.Index}
                  </span>
                </td>
                <td className="px-4 py-3 whitespace-nowrap">
                  <div className="flex items-center">
                    <svg className="w-4 h-4 text-gray-400 mr-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                    </svg>
                    <span className="font-mono text-xs text-gray-600 truncate max-w-[200px]">{block.Hash}</span>
                  </div>
                </td>
                <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                  {new Date(block.Timestamp).toLocaleString()}
                </td>
                <td className="px-4 py-3 whitespace-nowrap text-sm">
                  <span className="font-mono text-xs text-gray-600 truncate max-w-[100px]">{block.Validator}</span>
                </td>
                <td className="px-4 py-3 whitespace-nowrap">
                  <span className="px-2 py-1 text-xs font-medium bg-blue-100 text-blue-800 rounded-full">
                    {block.Transactions?.length || 0} transactions
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Block Detail Modal */}
      {selectedBlock && (
        <div className="fixed inset-0 flex items-center justify-center z-50">
          <div className="fixed inset-0 bg-black opacity-50" onClick={closeBlockDetails}></div>
          <div className="bg-white rounded-lg shadow-lg overflow-hidden max-w-4xl w-full mx-4 z-10 max-h-[90vh] overflow-y-auto">
            <div className="bg-blue-600 text-white px-6 py-4 flex justify-between items-center">
              <h3 className="text-xl font-bold">Block #{selectedBlock.Index} Details</h3>
              <button onClick={closeBlockDetails} className="text-white hover:text-gray-200 focus:outline-none">
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            <div className="p-6">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-6">
                <div>
                  <h4 className="font-semibold text-gray-700 mb-1">Hash</h4>
                  <p className="font-mono text-sm text-gray-600 break-all">{selectedBlock.Hash}</p>
                </div>
                <div>
                  <h4 className="font-semibold text-gray-700 mb-1">Previous Block Hash</h4>
                  <p className="font-mono text-sm text-gray-600 break-all">{selectedBlock.PrevHash}</p>
                </div>
                <div>
                  <h4 className="font-semibold text-gray-700 mb-1">Time</h4>
                  <p className="text-sm text-gray-600">{new Date(selectedBlock.Timestamp).toLocaleString()}</p>
                </div>
                <div>
                  <h4 className="font-semibold text-gray-700 mb-1">Validator</h4>
                  <p className="font-mono text-sm text-gray-600 break-all">{selectedBlock.Validator}</p>
                </div>
                <div>
                  <h4 className="font-semibold text-gray-700 mb-1">Human Proof</h4>
                  <p className="font-mono text-sm text-gray-600 break-all">{selectedBlock.HumanProof}</p>
                </div>
                <div>
                  <h4 className="font-semibold text-gray-700 mb-1">Nonce</h4>
                  <p className="text-sm text-gray-600">{selectedBlock.Nonce}</p>
                </div>
              </div>

              <h4 className="font-bold text-lg text-gray-800 mb-3">Transactions ({selectedBlock.Transactions?.length || 0})</h4>
              
              {selectedBlock.Transactions && selectedBlock.Transactions.length > 0 ? (
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-gray-200">
                    <thead className="bg-gray-50">
                      <tr>
                        <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">ID</th>
                        <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Sender</th>
                        <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Recipient</th>
                        <th className="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Amount</th>
                      </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-gray-200">
                      {selectedBlock.Transactions.map((tx: Transaction, index: number) => (
                        <tr key={tx.ID || `tx-${index}`}>
                          <td className="px-4 py-2 whitespace-nowrap text-sm text-gray-500 font-mono">
                            {tx.ID ? (tx.ID.substring(0, 10) + '...') : 'N/A'}
                          </td>
                          <td className="px-4 py-2 whitespace-nowrap text-sm text-gray-500 font-mono">
                            {tx.From ? (tx.From.substring(0, 10) + '...') : 'N/A'}
                          </td>
                          <td className="px-4 py-2 whitespace-nowrap text-sm text-gray-500 font-mono">
                            {tx.To ? (tx.To.substring(0, 10) + '...') : 'N/A'}
                          </td>
                          <td className="px-4 py-2 whitespace-nowrap text-sm text-gray-500">
                            {tx.Value || '0'} token
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ) : (
                <div className="text-center py-4 text-gray-500">
                  <p>No transactions in this block.</p>
                </div>
              )}
            </div>
            <div className="bg-gray-50 px-6 py-4 flex justify-end">
              <button 
                onClick={closeBlockDetails}
                className="bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
} 