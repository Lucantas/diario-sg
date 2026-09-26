ALTER TABLE cgu_sanctions DROP CONSTRAINT cgu_sanctions_register_check;
ALTER TABLE cgu_sanctions ADD CONSTRAINT cgu_sanctions_register_check CHECK (register IN ('CEIS', 'CNEP', 'CEPIM'));
