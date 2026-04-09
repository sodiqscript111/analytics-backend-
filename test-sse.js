#!/usr/bin/env node

const http = require('http');

console.log('Starting SSE test client...');
console.log('Connecting to http://localhost:8080/events/stream\n');

const url = new URL('http://localhost:8080/events/stream');
const options = {
  method: 'GET',
  headers: {
    'Accept': 'text/event-stream',
    'Cache-Control': 'no-cache',
    'Connection': 'keep-alive'
  }
};

const req = http.request(url, options, (res) => {
  console.log(`Status: ${res.statusCode}`);
  console.log(`Headers:`, res.headers);
  console.log('\n--- SSE Stream Started ---\n');

  let dataBuffer = '';

  res.on('data', (chunk) => {
    dataBuffer += chunk.toString();
    const lines = dataBuffer.split('\n');
    
    // Process complete lines
    for (let i = 0; i < lines.length - 1; i++) {
      const line = lines[i].trim();
      if (line) {
        console.log(`[${new Date().toISOString()}] ${line}`);
      }
    }
    
    // Keep incomplete line in buffer
    dataBuffer = lines[lines.length - 1];
  });

  res.on('end', () => {
    console.log('\n--- SSE Stream Ended ---');
  });

  res.on('error', (err) => {
    console.error('Stream error:', err);
  });
});

req.on('error', (err) => {
  console.error('Request error:', err);
});

// Keep the request open
console.log('Press Ctrl+C to stop the client\n');

setTimeout(() => {
  // Auto-disconnect after 30 seconds for testing
  console.log('\n--- Auto-disconnecting after 30 seconds ---');
  req.end();
  process.exit(0);
}, 30000);

req.end();
