import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  stages: [
    { duration: '2m', target: 500 },
    { duration: '30m', target: 500 },
    { duration: '2m', target: 0 },
  ],
  thresholds: {
    'http_req_duration': ['p(95)<500', 'p(99)<1000'],
    'http_req_failed': ['rate<0.01'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export function setup() {
  console.log('=== SOAK TEST: Creating test wallet ===');
  console.log('This test will run for ~34 minutes');
  
  let createRes = http.post(`${BASE_URL}/api/v1/wallet/create`);
  if (createRes.status !== 201) {
    throw new Error(`Failed to create wallet: ${createRes.status}`);
  }
  
  let wallet = JSON.parse(createRes.body);
  console.log(`Wallet created: ${wallet.walletId}`);
  
  let depositPayload = JSON.stringify({
    walletId: wallet.walletId,
    operationType: 'DEPOSIT',
    amount: 5000000.00
  });
  
  http.post(
    `${BASE_URL}/api/v1/wallet`,
    depositPayload,
    { headers: { 'Content-Type': 'application/json' } }
  );
  
  console.log('Initial deposit: 5000000.00');
  console.log('=== Starting soak test - checking for memory leaks ===');
  
  return { 
    walletId: wallet.walletId,
    startTime: new Date().toISOString()
  };
}

export default function (data) {
  let operation = Math.random() < 0.5 ? 'DEPOSIT' : 'WITHDRAW';
  let amount = Math.floor(Math.random() * 100) + 1;
  
  let payload = JSON.stringify({
    walletId: data.walletId,
    operationType: operation,
    amount: amount
  });
  
  let res = http.post(
    `${BASE_URL}/api/v1/wallet`,
    payload,
    { headers: { 'Content-Type': 'application/json' } }
  );
  
  check(res, {
    'status is 200': (r) => r.status === 200,
    'no 5xx errors': (r) => r.status < 500,
    'response time < 1000ms': (r) => r.timings.duration < 1000,
  });
  
  sleep(0.1);
}

export function teardown(data) {
  console.log('=== SOAK TEST COMPLETED ===');
  console.log(`Started at: ${data.startTime}`);
  console.log(`Ended at: ${new Date().toISOString()}`);
  
  let res = http.get(`${BASE_URL}/api/v1/wallets/${data.walletId}`);
  
  if (res.status === 200) {
    let balance = JSON.parse(res.body);
    console.log(`Final balance: ${balance.balance}`);
  }
}

