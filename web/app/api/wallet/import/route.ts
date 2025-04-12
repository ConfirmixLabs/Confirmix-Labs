import { NextResponse } from 'next/server';

const BACKEND_API_URL = process.env.NEXT_PUBLIC_BACKEND_API_URL || 'http://localhost:8080/api';

export async function OPTIONS() {
  return new NextResponse(null, {
    headers: {
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'POST, OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type',
    },
  });
}

export async function POST(request: Request) {
  console.log('Wallet import endpoint called');
  
  try {
    // Parse request body to get the private key
    const body = await request.json();
    console.log('Request body:', { privateKeyLength: body.privateKey?.length });
    
    const { privateKey } = body;
    
    if (!privateKey) {
      console.log('No private key provided in request');
      return NextResponse.json(
        { error: 'Private key is required' },
        { status: 400 }
      );
    }
    
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 5000);
    
    console.log('Sending request to backend:', `${BACKEND_API_URL}/wallet/import`);
    
    const response = await fetch(`${BACKEND_API_URL}/wallet/import`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ privateKey }),
      signal: controller.signal
    });
    
    clearTimeout(timeoutId);
    
    if (!response.ok) {
      const errorText = await response.text();
      console.error('Backend wallet import error:', {
        status: response.status,
        statusText: response.statusText,
        errorText
      });
      return NextResponse.json(
        { error: `Could not import wallet (${response.status}: ${response.statusText}) - ${errorText}` },
        { status: response.status }
      );
    }
    
    const data = await response.json();
    console.log('Backend wallet import response:', data);
    
    return NextResponse.json(data);
    
  } catch (error: any) {
    console.error('Wallet import error:', {
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
      { error: 'Could not import wallet: ' + error.message },
      { status: 500 }
    );
  }
} 