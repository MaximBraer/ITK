import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  stages: [
    { duration: '1m', target: 500 },
    { duration: '1m', target: 1000 },
    { duration: '1m', target: 2000 },
    { duration: '1m', target: 3000 },
    { duration: '2m', target: 3000 },
    { duration: '1m', target: 0 },
  ],
  thresholds: {
    'http_req_duration': ['p(99)<2000'],
    'http_req_failed': ['rate<0.10'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export function setup() {
  console.log('=== STRESS TEST: Creating test wallet ===');
  
  let createRes = http.post(`${BASE_URL}/api/v1/wallet/create`);
  if (createRes.status !== 201) {
    throw new Error(`Failed to create wallet: ${createRes.status}`);
  }
  
  let wallet = JSON.parse(createRes.body);
  console.log(`Wallet created: ${wallet.walletId}`);
  
  let depositPayload = JSON.stringify({
    walletId: wallet.walletId,
    operationType: 'DEPOSIT',
    amount: 1000000.00
  });
  
  http.post(
    `${BASE_URL}/api/v1/wallet`,
    depositPayload,
    { headers: { 'Content-Type': 'application/json' } }
  );
  
  console.log('Initial deposit: 1000000.00');
  console.log('=== Starting stress test - finding system limits ===');
  
  return { walletId: wallet.walletId };
}

export default function (data) {
  let operation = Math.random() < 0.8 ? 'DEPOSIT' : 'WITHDRAW';
  let amount = Math.floor(Math.random() * 20) + 1;
  
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
    'no 5xx errors': (r) => r.status < 500,
    'response received': (r) => r.status !== 0,
  });
  
  sleep(0.01);
}

export function teardown(data) {
  console.log('=== STRESS TEST COMPLETED ===');
  
  let res = http.get(`${BASE_URL}/api/v1/wallets/${data.walletId}`);
  
  if (res.status === 200) {
    let balance = JSON.parse(res.body);
    console.log(`Final balance: ${balance.balance}`);
  }
}

