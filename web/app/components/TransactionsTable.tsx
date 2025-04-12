'use client';

import { Transaction } from '../lib/types';

interface TransactionsTableProps {
  transactions: Transaction[];
  loading?: boolean;
}

export default function TransactionsTable({ transactions, loading = false }: TransactionsTableProps) {
  if (loading) {
    return (
      <div className="bg-white shadow rounded-lg p-4">
        <div className="animate-pulse space-y-4">
          <div className="h-4 bg-gray-200 rounded w-1/4"></div>
          <div className="space-y-3">
            <div className="h-4 bg-gray-200 rounded"></div>
            <div className="h-4 bg-gray-200 rounded"></div>
            <div className="h-4 bg-gray-200 rounded"></div>
          </div>
        </div>
      </div>
    );
  }

  if (transactions.length === 0) {
    return (
      <div className="bg-white shadow rounded-lg p-8 text-center">
        <div className="text-center py-4">
          <svg className="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
          </svg>
          <h3 className="mt-2 text-sm font-medium text-gray-900">No Transactions Found</h3>
          <p className="mt-1 text-sm text-gray-500">No transactions have been made yet.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white shadow rounded-lg overflow-hidden">
      <div className="px-4 py-5 sm:px-6">
        <h3 className="text-lg leading-6 font-medium text-gray-900">Recent Transactions</h3>
      </div>
      <div className="border-t border-gray-200">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Sender
              </th>
              <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Recipient
              </th>
              <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Amount
              </th>
              <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Status
              </th>
              <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Date
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {transactions.map((tx) => (
              <tr key={tx.ID}>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {tx.From ? `${tx.From.substring(0, 10)}...` : 'N/A'}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {tx.To ? `${tx.To.substring(0, 10)}...` : 'N/A'}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {formatConxAmount(tx.Value)} ConX
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm">
                  {tx.Status === "confirmed" ? (
                    <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-green-100 text-green-800">
                      Confirmed
                    </span>
                  ) : tx.Status === "pending" ? (
                    <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-yellow-100 text-yellow-800">
                      Pending
                    </span>
                  ) : (
                    <span className="px-2 inline-flex text-xs leading-5 font-semibold rounded-full bg-gray-100 text-gray-800">
                      Unknown
                    </span>
                  )}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {tx.Timestamp ? new Date(tx.Timestamp).toLocaleString() : 'N/A'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// Helper function to format ConX amounts
function formatConxAmount(amountInSmallestUnit: number | string): string {
  if (!amountInSmallestUnit) return '0';
  
  // Convert to BigInt to handle large numbers accurately
  const amount = typeof amountInSmallestUnit === 'string' 
    ? BigInt(amountInSmallestUnit) 
    : BigInt(amountInSmallestUnit);
  
  // Convert to string and pad with leading zeros if needed
  let amountStr = amount.toString();
  
  // If the amount is less than 1 ConX
  if (amountStr.length <= 18) {
    amountStr = amountStr.padStart(19, '0');
    return '0.' + amountStr.substring(0, 18).replace(/0+$/, '') || '0';
  }
  
  // Insert decimal point 18 places from the right
  const decimalIndex = amountStr.length - 18;
  const integerPart = amountStr.substring(0, decimalIndex);
  const fractionalPart = amountStr.substring(decimalIndex, decimalIndex + 4); // Show only first 4 decimal places for readability
  
  // Format with commas for thousands separator
  const formattedIntegerPart = parseInt(integerPart).toLocaleString();
  
  return fractionalPart === '0000' 
    ? formattedIntegerPart 
    : `${formattedIntegerPart}.${fractionalPart}`;
} 