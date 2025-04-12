import { NextResponse } from 'next/server';

const BACKEND_API_URL = process.env.NEXT_PUBLIC_BACKEND_API_URL || 'http://localhost:8080/api';

export async function GET() {
  try {
    const response = await fetch(`${BACKEND_API_URL}/transactions/confirmed`);
    if (!response.ok) {
      return NextResponse.json(
        { error: 'Could not retrieve confirmed transaction list' },
        { status: response.status }
      );
    }
    return NextResponse.json(await response.json());
  } catch (error) {
    console.error('Confirmed transactions endpoint error:', error);
    return NextResponse.json(
      { error: 'Onaylanmış işlem listesi alınamadı' },
      { status: 500 }
    );
  }
} 