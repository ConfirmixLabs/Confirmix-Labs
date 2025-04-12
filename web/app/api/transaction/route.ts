import { NextResponse } from 'next/server';

const BACKEND_API_URL = process.env.NEXT_PUBLIC_BACKEND_API_URL || 'http://localhost:8080/api';

export async function POST(request: Request) {
  try {
    console.log('Received transaction request');
    const body = await request.json();
    console.log('Request body:', body);
    
    console.log('Sending request to backend:', `${BACKEND_API_URL}/wallet/transfer`);
    const response = await fetch(`${BACKEND_API_URL}/wallet/transfer`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    });

    console.log('Backend response status:', response.status);
    console.log('Backend response headers:', Object.fromEntries(response.headers.entries()));

    let responseData;
    try {
      const responseText = await response.text();
      console.log('Raw response from backend:', responseText);
      responseData = JSON.parse(responseText);
    } catch (error) {
      console.error('Error parsing response JSON:', error);
      return NextResponse.json(
        { error: 'Invalid response from server' },
        { status: 500 }
      );
    }

    if (!response.ok) {
      console.error('Backend returned error:', responseData);
      return NextResponse.json(
        { error: responseData.error || 'Transfer transaction failed' },
        { status: response.status }
      );
    }

    console.log('Transaction successful:', responseData);
    return NextResponse.json(responseData, { status: 200 });
  } catch (error) {
    console.error('Transaction API error:', error);
    return NextResponse.json(
      { error: 'An error occurred during the transfer transaction' },
      { status: 500 }
    );
  }
} 