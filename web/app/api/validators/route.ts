import { NextResponse } from 'next/server';

const BACKEND_API_URL = process.env.NEXT_PUBLIC_BACKEND_API_URL || 'http://localhost:8080/api';

export async function GET() {
  console.log('Validators endpoint called, attempting to connect to:', BACKEND_API_URL);
  
  try {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 5000); // 5 saniye timeout

    const response = await fetch(`${BACKEND_API_URL}/validators`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
      signal: controller.signal
    });

    clearTimeout(timeoutId);

    if (!response.ok) {
      console.error('Backend validators error:', response.status, response.statusText);
      return NextResponse.json(
        { error: `Validatör verileri alınamadı (${response.status}: ${response.statusText})` },
        { status: response.status }
      );
    }

    const data = await response.json();
    console.log('Backend validators response:', JSON.stringify(data, null, 2));

    // Veri yapısını kontrol et ve düzelt
    const validators = Array.isArray(data) ? data : [];
    
    // Her validatörün gerekli alanları var mı kontrol et
    const formattedValidators = validators.map(validator => ({
      address: validator.address || '',
      humanProof: validator.humanProof || '',
      registrationTime: validator.registrationTime || Date.now()
    }));

    return NextResponse.json(formattedValidators);

  } catch (error: any) {
    console.error('Validators fetch error:', {
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
      { error: 'Validatör verileri alınamadı: ' + error.message },
      { status: 500 }
    );
  }
} 