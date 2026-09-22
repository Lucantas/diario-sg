-- Edição extraordinária: o site publica a extra do mesmo dia em
-- diario/AAAA_MM_DD_N.pdf (ver services/scraper, adapter pmsg).
ALTER TABLE gazettes ADD COLUMN is_extra boolean
    GENERATED ALWAYS AS (source_url ~ '_[0-9]{2}_[0-9]{2}_[0-9]+\.pdf$') STORED;
