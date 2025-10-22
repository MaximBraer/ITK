import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  vus: 100,
  duration: '2m',
  thresholds: {
    'http_req_duration': ['p(95)<300'],
    'http_req_failed': ['rate<0.01'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export function setup() {
  console.log('=== MULTI-WALLET TEST: Creating 10 test wallets ===');
  
  let wallets = [];
  
  for (let i = 0; i < 10; i++) {
    let createRes = http.post(`${BASE_URL}/api/v1/wallet/create`);
    if (createRes.status !== 201) {
      throw new Error(`Failed to create wallet ${i}: ${createRes.status}`);
    }
    
    let wallet = JSON.parse(createRes.body);
    
    let depositPayload = JSON.stringify({
      walletId: wallet.walletId,
      operationType: 'DEPOSIT',
      amount: 50000.00
    });
    
    http.post(
      `${BASE_URL}/api/v1/wallet`,
      depositPayload,
      { headers: { 'Content-Type': 'application/json' } }
    );
    
    wallets.push(wallet.walletId);
    console.log(`Wallet ${i + 1}/10 created: ${wallet.walletId}`);
  }
  
  console.log('=== All wallets created. Starting multi-wallet test ===');
  
  return { wallets: wallets };
}

export default function (data) {
  let walletId = data.wallets[Math.floor(Math.random() * data.wallets.length)];
  
  let operation = Math.random() < 0.6 ? 'DEPOSIT' : 'WITHDRAW';
  let amount = Math.floor(Math.random() * 50) + 1;
  
  let payload = JSON.stringify({
    walletId: walletId,
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
  });
  
  sleep(0.05);
}

export function teardown(data) {
  console.log('=== MULTI-WALLET TEST COMPLETED ===');
  console.log('Checking final balances...');
  
  data.wallets.forEach((walletId, index) => {
    let res = http.get(`${BASE_URL}/api/v1/wallets/${walletId}`);
    if (res.status === 200) {
      let balance = JSON.parse(res.body);
      console.log(`Wallet ${index + 1} balance: ${balance.balance}`);
    }
  });
}

