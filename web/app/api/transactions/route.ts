import { NextResponse } from 'next/server';

const BACKEND_API_URL = process.env.NEXT_PUBLIC_BACKEND_API_URL || 'http://localhost:8080/api';

export async function GET() {
  try {
    const response = await fetch(`${BACKEND_API_URL}/transactions`);
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    return NextResponse.json(await response.json());
  } catch (error) {
    console.error('Transactions endpoint error:', error);
    return NextResponse.json(
      { error: 'İşlem listesi alınamadı' },
      { status: 500 }
    );
  }
} 