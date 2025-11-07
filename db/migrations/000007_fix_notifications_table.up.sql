-- Add missing columns to notifications table
ALTER TABLE notifications
ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'pending'
CHECK (status IN ('pending', 'sent', 'failed', 'scheduled'));

ALTER TABLE notifications
ADD COLUMN IF NOT EXISTS scheduled_at TIMESTAMP WITH TIME ZONE;

ALTER TABLE notifications
ADD COLUMN IF NOT EXISTS error_message TEXT;

ALTER TABLE notifications
ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP;

-- Add index for status column (for efficient queries)
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);

-- Add index for scheduled notifications query
CREATE INDEX IF NOT EXISTS idx_notifications_scheduled ON notifications(status, scheduled_at)
WHERE status = 'scheduled' OR status = 'pending';

-- Add trigger for updated_at
CREATE TRIGGER update_notifications_updated_at BEFORE UPDATE ON notifications
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
