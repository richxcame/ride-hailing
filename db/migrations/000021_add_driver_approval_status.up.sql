-- Add driver approval status fields
-- This separates approval status from availability status

ALTER TABLE drivers
ADD COLUMN approval_status VARCHAR(20) DEFAULT 'pending' CHECK (approval_status IN ('pending', 'approved', 'rejected')),
ADD COLUMN approved_by UUID REFERENCES users(id),
ADD COLUMN approved_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN rejection_reason TEXT,
ADD COLUMN rejected_at TIMESTAMP WITH TIME ZONE;

-- Create index for filtering by approval status
CREATE INDEX idx_drivers_approval_status ON drivers(approval_status);

-- Migrate existing data: if is_available = true, consider approved
-- This assumes all current drivers with is_available=true are approved
UPDATE drivers
SET approval_status = 'approved',
    approved_at = created_at
WHERE is_available = true;

-- Reset is_available for all drivers (it will be set when they go online)
UPDATE drivers SET is_available = false;

-- Add comments
COMMENT ON COLUMN drivers.approval_status IS 'Driver approval status by admin: pending (awaiting approval), approved (can go online), rejected (denied)';
COMMENT ON COLUMN drivers.is_available IS 'Driver current availability for accepting rides (set by driver going online/offline)';
COMMENT ON COLUMN drivers.is_online IS 'Driver is currently online and actively accepting rides';
