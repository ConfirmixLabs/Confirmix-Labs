import './globals.css';
import type { Metadata } from 'next';
import ErrorBoundary from './components/ErrorBoundary';
import { Inter } from 'next/font/google';
import ApiStatus from './components/ApiStatus';
import { BlockchainProvider } from './contexts/BlockchainContext';

const inter = Inter({ subsets: ['latin'] });

export const metadata: Metadata = {
  title: 'Confirmix Blockchain',
  description: 'Hybrid PoA-PoH Blockchain',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className={`${inter.className} bg-gray-100 min-h-screen`}>
        <ErrorBoundary>
          <BlockchainProvider>
            {children}
          </BlockchainProvider>
          <ApiStatus />
        </ErrorBoundary>
      </body>
    </html>
  );
} 