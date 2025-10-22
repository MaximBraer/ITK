CREATE TABLE operations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    operation_type VARCHAR(20) NOT NULL CHECK (operation_type IN ('DEPOSIT', 'WITHDRAW')),
    amount NUMERIC(20, 2) NOT NULL CHECK (amount > 0),
    balance_after NUMERIC(20, 2) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_operations_wallet_id ON operations(wallet_id);
CREATE INDEX idx_operations_created_at ON operations(created_at DESC);

