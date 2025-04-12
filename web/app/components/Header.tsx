'use client';

type HeaderProps = {
  status: {
    height: number;
    lastBlock: any;
  } | null;
  transactionCount: number;
  validatorCount: number;
  usingMockData?: boolean;
  onRefresh: () => void;
};

export default function Header({ 
  status, 
  transactionCount, 
  validatorCount, 
  usingMockData, 
  onRefresh 
}: HeaderProps) {
  // To get the last block hash
  const getLastBlockHash = (lastBlock: any): string => {
    if (typeof lastBlock === 'string') {
      try {
        return JSON.parse(lastBlock).hash || lastBlock.substring(0, 15);
      } catch (e) {
        return lastBlock.substring(0, 15);
      }
    } else if (lastBlock && typeof lastBlock === 'object') {
      return lastBlock.hash;
    }
    return 'Unknown';
  };
  
  return (
    <header>
      {usingMockData && (
        <div className="bg-yellow-100 text-yellow-800 px-4 py-2 text-sm text-center mb-4 rounded-lg border border-yellow-300">
          <div className="flex items-center justify-center">
            <svg className="w-5 h-5 mr-2 text-yellow-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
            <span>There is a problem accessing the API server. The data shown may not be up to date.</span>
            <button 
              onClick={onRefresh} 
              className="ml-3 px-2 py-1 bg-yellow-200 hover:bg-yellow-300 rounded-md text-xs font-medium"
            >
              Refresh
            </button>
          </div>
        </div>
      )}
      
      <div className="bg-gradient-to-r from-blue-600 to-indigo-700 text-white rounded-xl shadow-lg p-6 mb-6">
        <div className="flex items-center justify-between mb-4">
          <h1 className="text-2xl text-white font-bold">Confirmix Blockchain</h1>
        
        </div>
        
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {/* Card 1 - Blockchain Height */}
          <div className="bg-white rounded-lg shadow-md hover:shadow-lg transition-all duration-200 p-4">
            <div className="flex items-center">
              <div className="bg-blue-600 rounded-lg p-3 mr-3 flex-shrink-0">
                <svg className="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                </svg>
              </div>
              <div>
                <h3 className="text-xs font-medium text-gray-500 uppercase tracking-wide">Blockchain Height</h3>
                <p className="text-sm font-mono truncate max-w-[150px] text-gray-900">{status?.height || '...'}</p>
              </div>
            </div>
          </div>
          
          {/* Card 2 - Latest Block */}
          <div className="bg-white rounded-lg shadow-md hover:shadow-lg transition-all duration-200 p-4">
            <div className="flex items-center">
              <div className="bg-indigo-600 rounded-lg p-3 mr-3 flex-shrink-0">
                <svg className="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
              <div>
                <h3 className="text-xs font-medium text-gray-500 uppercase tracking-wide">Latest Block</h3>
                <p className="text-sm font-mono truncate max-w-[150px] text-gray-900">
                  {status ? getLastBlockHash(status.lastBlock) : '...'}
                </p>
              </div>
            </div>
          </div>
          
          {/* Card 3 - Pending Transactions */}
          <div className="bg-white rounded-lg shadow-md hover:shadow-lg transition-all duration-200 p-4">
            <div className="flex items-center">
              <div className="bg-purple-600 rounded-lg p-3 mr-3 flex-shrink-0">
                <svg className="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
              </div>
              <div>
                <h3 className="text-xs font-medium text-gray-500 uppercase tracking-wide">Pending Tx</h3>
                <p className="text-sm font-mono truncate max-w-[150px] text-gray-900">{transactionCount}</p>
              </div>
            </div>
          </div>
          
          {/* Card 4 - Validators */}
          <div className="bg-white rounded-lg shadow-md hover:shadow-lg transition-all duration-200 p-4">
            <div className="flex items-center">
              <div className="bg-green-600 rounded-lg p-3 mr-3 flex-shrink-0">
                <svg className="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                </svg>
              </div>
              <div>
                <h3 className="text-xs font-medium text-gray-500 uppercase tracking-wide">Validators</h3>
                <p className="text-sm font-mono truncate max-w-[150px] text-gray-900">{validatorCount}</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </header>
  );
} 