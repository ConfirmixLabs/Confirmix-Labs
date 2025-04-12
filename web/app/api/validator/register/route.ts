import { NextResponse } from 'next/server';

const BACKEND_API_URL = process.env.NEXT_PUBLIC_BACKEND_API_URL || 'http://localhost:8080/api';

export async function POST(request: Request) {
  console.log('Validator registration endpoint called');
  
  try {
    const body = await request.json();
    const { address, humanProof } = body;

    if (!address || !humanProof) {
      return NextResponse.json(
        { error: 'Adres ve humanProof alanları zorunludur' },
        { status: 400 }
      );
    }

    console.log('Attempting to register validator:', { address, humanProof });

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 5000);

    const response = await fetch(`${BACKEND_API_URL}/validator/register`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ address, humanProof }),
      signal: controller.signal
    });

    clearTimeout(timeoutId);

    if (!response.ok) {
      console.error('Backend validator registration error:', response.status, response.statusText);
      return NextResponse.json(
        { error: `Validator kaydı başarısız (${response.status}: ${response.statusText})` },
        { status: response.status }
      );
    }

    const data = await response.json();
    console.log('Backend validator registration response:', data);

    return NextResponse.json(data);

  } catch (error: any) {
    console.error('Validator registration error:', {
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
      { error: 'Validator kaydı başarısız: ' + error.message },
      { status: 500 }
    );
  }
} 