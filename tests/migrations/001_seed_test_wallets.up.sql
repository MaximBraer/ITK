INSERT INTO wallets (id, balance, created_at, updated_at)
VALUES 
    ('550e8400-e29b-41d4-a716-446655440001', 5000.00, NOW(), NOW()),
    ('550e8400-e29b-41d4-a716-446655440002', 1000.50, NOW(), NOW()),
    ('550e8400-e29b-41d4-a716-446655440003', 0.00, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

