import { NextResponse } from 'next/server';

const BACKEND_API_URL = process.env.NEXT_PUBLIC_BACKEND_API_URL || 'http://localhost:8080/api';

export async function GET() {
  try {
    const response = await fetch(`${BACKEND_API_URL}/status`);
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    return NextResponse.json(await response.json());
  } catch (error) {
    console.error('Status endpoint error:', error);
    return NextResponse.json(
      { error: 'Blockchain durumu alınamadı' },
      { status: 500 }
    );
  }
} 