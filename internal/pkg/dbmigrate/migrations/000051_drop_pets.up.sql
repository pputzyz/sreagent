-- Migration 000051: Drop unused pets feature tables.
-- pet_interactions and pets were experimental gamification tables that are no longer used.
-- Down migration restores the tables from backup if needed.
DROP TABLE IF EXISTS pet_interactions, pets;
