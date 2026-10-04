-- new users must opt in to the daily digest; the previous default opted them in
-- silently for any insert that omitted the column
ALTER TABLE preferences ALTER COLUMN digest_enabled SET DEFAULT false;
