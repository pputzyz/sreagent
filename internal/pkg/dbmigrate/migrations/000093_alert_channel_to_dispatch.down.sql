-- Remove dispatch_policies that were migrated from alert_channels
DELETE FROM dispatch_policies WHERE description LIKE 'Migrated from alert_channel #%';
