'use client';

import { useState, useEffect } from 'react';

export default function ApiStatus() {
  const [isConnected, setIsConnected] = useState<boolean | null>(null);
  const [isChecking, setIsChecking] = useState(false);
  const [retryCount, setRetryCount] = useState(0);
  const [lastChecked, setLastChecked] = useState<Date | null>(null);

  const checkConnection = async () => {
    if (isChecking) return;
    
    setIsChecking(true);
    try {
      // Sadece OPTIONS istekleriyle kontrol et
      // Daha hafif bir istek türü
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 3000); // 3 saniye timeout
      
      const response = await fetch('/api/status', {
        method: 'OPTIONS',
        signal: controller.signal,
        cache: 'no-store',
        headers: {
          'Cache-Control': 'no-cache'
        }
      });
      
      clearTimeout(timeoutId);
      setIsConnected(response.ok);
      setLastChecked(new Date());
      setRetryCount(0); // Başarılı kontrolde retry sayısını sıfırla
    } catch (error) {
      console.error('API bağlantı kontrolü sırasında hata:', error);
      setIsConnected(false);
      setLastChecked(new Date());
      setRetryCount(prev => prev + 1);
    } finally {
      setIsChecking(false);
    }
  };

  useEffect(() => {
    checkConnection();
    
    // Başarısız kontroller için artan gecikmeyle yeniden dene
    const getRetryDelay = () => {
      if (retryCount === 0) return 30000; // İlk kontrol başarılıysa 30 saniye sonra tekrar kontrol et
      return Math.min(30000 * Math.pow(1.5, retryCount - 1), 300000); // Max 5 dakika
    };
    
    const interval = setInterval(checkConnection, getRetryDelay());
    return () => clearInterval(interval);
  }, [retryCount]);

  if (isConnected === null) {
    return (
      <div className="fixed bottom-4 right-4 p-2 bg-gray-200 rounded-full shadow-md">
        <svg className="animate-spin h-5 w-5 text-gray-600" fill="none" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
      </div>
    );
  }

  return (
    <div 
      className={`fixed bottom-4 right-4 px-3 py-2 rounded-md shadow-md flex items-center space-x-2 ${
        isConnected ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
      }`}
    >
      <div 
        className={`h-3 w-3 rounded-full ${
          isConnected ? 'bg-green-600' : 'bg-red-600'
        }`}
      ></div>
      <span className="text-sm font-medium">
        {isConnected ? 'API Bağlı' : 'API Bağlantı Hatası'}
      </span>
      {!isConnected && (
        <button
          onClick={checkConnection}
          disabled={isChecking}
          className="ml-2 px-2 py-1 text-xs bg-red-200 hover:bg-red-300 rounded"
        >
          {isChecking ? 'Kontrol ediliyor...' : 'Tekrar Dene'}
        </button>
      )}
      {lastChecked && (
        <span className="text-xs opacity-70 ml-2">
          {new Intl.DateTimeFormat('tr-TR', {
            hour: '2-digit',
            minute: '2-digit',
            second: '2-digit',
          }).format(lastChecked)}
        </span>
      )}
    </div>
  );
} 