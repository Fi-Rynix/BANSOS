-- Peserta dan pengelola khusus uji beban war. Id dipatok mulai 900000 supaya
-- token stub bisa dibuat tanpa membaca database, dan jauh dari id hasil SERIAL.
-- Jalankan: docker compose exec -T postgres psql -U postgres -d bansos_backend < tests/loadtest/seed_users.sql
INSERT INTO users (id, name, email, role)
VALUES (900000, 'Pengelola Uji Beban', 'pengelola-war@bansos.test', 'pengelola')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (id, name, email, role)
SELECT 900000 + n, 'Peserta War ' || n, 'peserta-war-' || n || '@bansos.test', 'peserta'
FROM generate_series(1, 500) AS n
ON CONFLICT (id) DO NOTHING;
