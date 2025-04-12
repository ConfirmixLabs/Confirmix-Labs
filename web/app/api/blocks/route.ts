import { NextResponse } from 'next/server';

const BACKEND_API_URL = process.env.NEXT_PUBLIC_BACKEND_API_URL || 'http://localhost:8080/api';

export async function GET() {
  try {
    const response = await fetch(`${BACKEND_API_URL}/blocks`);
    if (!response.ok) {
      return NextResponse.json(
        { error: 'Could not retrieve block list' },
        { status: response.status }
      );
    }
    return NextResponse.json(await response.json());
  } catch (error) {
    console.error('Blocks endpoint error:', error);
    return NextResponse.json(
      { error: 'Blok listesi alınamadı' },
      { status: 500 }
    );
  }
} 