import { NextResponse } from 'next/server';

const BACKEND_API_URL = process.env.NEXT_PUBLIC_BACKEND_API_URL || 'http://localhost:8080/api';

export async function POST() {
  console.log('Wallet creation endpoint called');
  
  try {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 5000);

    const response = await fetch(`${BACKEND_API_URL}/wallet/create`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      signal: controller.signal
    });

    clearTimeout(timeoutId);

    if (!response.ok) {
      console.error('Backend wallet creation error:', response.status, response.statusText);
      return NextResponse.json(
        { error: `Could not create wallet (${response.status}: ${response.statusText})` },
        { status: response.status }
      );
    }

    const data = await response.json();
    console.log('Backend wallet creation response:', data);

    return NextResponse.json(data);

  } catch (error: any) {
    console.error('Wallet creation error:', {
      name: error.name,
      message: error.message,
      cause: error.cause,
      code: error.cause?.code
    });

    if (error.name === 'AbortError') {
      return NextResponse.json(
        { error: 'Backend server did not respond (timeout)' },
        { status: 504 }
      );
    }

    if (error.cause?.code === 'ECONNREFUSED') {
      return NextResponse.json(
        { error: 'Could not connect to backend server. Please make sure the server is running.' },
        { status: 503 }
      );
    }

    return NextResponse.json(
      { error: 'Could not create wallet: ' + error.message },
      { status: 500 }
    );
  }
} 