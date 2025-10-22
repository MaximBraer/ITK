import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  stages: [
    { duration: '10s', target: 100 },
    { duration: '20s', target: 100 },
    { duration: '10s', target: 2000 },
    { duration: '30s', target: 2000 },
    { duration: '10s', target: 100 },
    { duration: '20s', target: 100 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    'http_req_duration': ['p(95)<1000'],
    'http_req_failed': ['rate<0.05'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export function setup() {
  console.log('=== SPIKE TEST: Creating test wallet ===');
  
  let createRes = http.post(`${BASE_URL}/api/v1/wallet/create`);
  if (createRes.status !== 201) {
    throw new Error(`Failed to create wallet: ${createRes.status}`);
  }
  
  let wallet = JSON.parse(createRes.body);
  console.log(`Wallet created: ${wallet.walletId}`);
  
  let depositPayload = JSON.stringify({
    walletId: wallet.walletId,
    operationType: 'DEPOSIT',
    amount: 500000.00
  });
  
  http.post(
    `${BASE_URL}/api/v1/wallet`,
    depositPayload,
    { headers: { 'Content-Type': 'application/json' } }
  );
  
  console.log('Initial deposit: 500000.00');
  console.log('=== Starting spike test ===');
  
  return { walletId: wallet.walletId };
}

export default function (data) {
  let operation = Math.random() < 0.7 ? 'DEPOSIT' : 'WITHDRAW';
  let amount = Math.floor(Math.random() * 50) + 1;
  
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
    'status is 200 or 409': (r) => r.status === 200 || r.status === 409,
    'no 5xx errors': (r) => r.status < 500,
  });
  
  sleep(0.01);
}

export function teardown(data) {
  console.log('=== SPIKE TEST COMPLETED ===');
  
  let res = http.get(`${BASE_URL}/api/v1/wallets/${data.walletId}`);
  
  if (res.status === 200) {
    let balance = JSON.parse(res.body);
    console.log(`Final balance: ${balance.balance}`);
  }
}

