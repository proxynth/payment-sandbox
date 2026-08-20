-- +goose Up

ALTER TABLE payments ADD COLUMN amount INTEGER NOT NULL;
ALTER TABLE payments ADD COLUMN currency TEXT NOT NULL;
ALTER TABLE payments ADD COLUMN authorized_amount INTEGER NOT NULL DEFAULT 0;
ALTER TABLE payments ADD COLUMN captured_amount INTEGER NOT NULL DEFAULT 0;
ALTER TABLE payments ADD COLUMN refunded_amount INTEGER NOT NULL DEFAULT 0;
-- +goose Down

ALTER TABLE payments DROP COLUMN amount;
ALTER TABLE payments DROP COLUMN currency;
ALTER TABLE payments DROP COLUMN authorized_amount;
ALTER TABLE payments DROP COLUMN captured_amount;
ALTER TABLE payments DROP COLUMN refunded_amount;