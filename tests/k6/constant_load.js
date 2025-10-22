import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';

const operationCounter = new Counter('wallet_operations');
const balanceErrorTrend = new Trend('balance_errors');

export let options = {
  vus: 1000,
  duration: '1m',
  thresholds: {
    'http_req_duration': ['p(95)<500', 'p(99)<1000'],
    'http_req_failed': ['rate<0.01'],
    'checks': ['rate>0.99'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export function setup() {
  console.log('Creating test wallet...');
  
  let createRes = http.post(`${BASE_URL}/api/v1/wallet/create`);
  check(createRes, {
    'wallet created': (r) => r.status === 201,
  });
  
  if (createRes.status !== 201) {
    throw new Error(`Failed to create wallet: ${createRes.status}`);
  }
  
  let wallet = JSON.parse(createRes.body);
  console.log(`Wallet created: ${wallet.walletId}`);
  
  let depositPayload = JSON.stringify({
    walletId: wallet.walletId,
    operationType: 'DEPOSIT',
    amount: 100000.00
  });
  
  let depositRes = http.post(
    `${BASE_URL}/api/v1/wallet`,
    depositPayload,
    { headers: { 'Content-Type': 'application/json' } }
  );
  
  check(depositRes, {
    'initial deposit successful': (r) => r.status === 200,
  });
  
  console.log(`Initial deposit completed. Starting load test...`);
  
  return { 
    walletId: wallet.walletId,
    initialBalance: 100000.00
  };
}

export default function (data) {
  let operation = Math.random() < 0.6 ? 'DEPOSIT' : 'WITHDRAW';
  let amount = Math.floor(Math.random() * 10) + 1;
  
  let payload = JSON.stringify({
    walletId: data.walletId,
    operationType: operation,
    amount: amount
  });
  
  let res = http.post(
    `${BASE_URL}/api/v1/wallet`,
    payload,
    { 
      headers: { 'Content-Type': 'application/json' },
      tags: { operation: operation }
    }
  );
  
  let success = check(res, {
    'status is 200': (r) => r.status === 200,
    'no 5xx errors': (r) => r.status < 500,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });
  
  if (success) {
    operationCounter.add(1);
  }
  
  sleep(0.01);
}

export function teardown(data) {
  console.log('Fetching final balance...');
  
  let res = http.get(`${BASE_URL}/api/v1/wallets/${data.walletId}`);
  
  if (res.status === 200) {
    let balance = JSON.parse(res.body);
    console.log(`Final balance: ${balance.balance}`);
    console.log(`Initial balance: ${data.initialBalance}`);
  } else {
    console.log(`Failed to get final balance: ${res.status}`);
  }
}

