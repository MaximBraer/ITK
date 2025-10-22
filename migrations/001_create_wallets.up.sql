CREATE TABLE wallets (
    id UUID PRIMARY KEY,
    balance NUMERIC(20, 2) NOT NULL DEFAULT 0 CHECK (balance >= 0),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wallets_created_at ON wallets(created_at DESC);
CREATE INDEX idx_wallets_balance ON wallets(balance);

