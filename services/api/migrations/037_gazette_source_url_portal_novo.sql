UPDATE gazettes
SET source_url = regexp_replace(source_url, '/diario/([0-9]{4}_[0-9]{2}_[0-9]{2}(_[0-9]+)?\.pdf)$', '/diario/download/pdf/\1')
WHERE source = 'diario_prefeitura'
  AND source_url ~ '^https://do\.pmsg\.rj\.gov\.br/diario/[0-9]{4}_[0-9]{2}_[0-9]{2}(_[0-9]+)?\.pdf$';
