-- Add soft delete support to payment_orders table
-- Migration: 049_add_payment_order_soft_delete.sql

ALTER TABLE payment_orders ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_payment_orders_deleted_at ON payment_orders(deleted_at);
