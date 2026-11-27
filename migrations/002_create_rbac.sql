CREATE TABLE IF NOT EXISTS roles (
    name        VARCHAR(20)  PRIMARY KEY,
    description VARCHAR(150) NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

INSERT INTO roles (name, description) VALUES
    ('admin',    'Akses penuh: kelola pengguna, modul, batch, verifikasi klaim, dan penerbitan sertifikat'),
    ('pengelola', 'Memverifikasi klaim, menerbitkan sertifikat, dan mengelola konten modul'),
    ('peserta',  'Mengikuti war slot, mengerjakan modul, kuis, dan mengunduh sertifikat sendiri')
ON CONFLICT (name) DO NOTHING;

CREATE TABLE IF NOT EXISTS permissions (
    name        VARCHAR(60)  PRIMARY KEY,
    description VARCHAR(150) NOT NULL
);

INSERT INTO permissions (name, description) VALUES
    ('module:list',          'Melihat katalog modul yang aktif'),
    ('module:manage',        'Membuat, mengubah, dan menonaktifkan modul'),
    ('batch:list',           'Melihat daftar batch beserta kuotanya'),
    ('batch:manage',         'Membuat batch, membuka, dan menutup jadwal'),
    ('claim:create',         'Mengambil slot atau masuk daftar tunggu'),
    ('claim:read:own',       'Melihat riwayat klaim sendiri'),
    ('claim:cancel',         'Membatalkan klaim sendiri'),
    ('claim:read:any',       'Melihat seluruh klaim peserta'),
    ('claim:verify',         'Menyetujui, menolak, atau memindahkan klaim ke daftar tunggu'),
    ('progress:read:own',    'Melihat progres belajar sendiri'),
    ('progress:write:own',   'Menandai lesson selesai dan mengirim skor kuis'),
    ('progress:read:any',    'Melihat progres belajar seluruh peserta'),
    ('certificate:read:own', 'Melihat dan mengunduh sertifikat sendiri'),
    ('certificate:issue',    'Menerbitkan dan mencabut sertifikat'),
    ('certificate:verify',   'Memverifikasi sertifikat lewat halaman publik tanpa login'),
    ('content:manage',       'Mengelola konten modul di MongoDB'),
    ('user:manage',          'Mengelola akun pengguna beserta rolenya')
ON CONFLICT (name) DO NOTHING;

CREATE TABLE IF NOT EXISTS role_permissions (
    role_name       VARCHAR(20) NOT NULL REFERENCES roles (name) ON DELETE CASCADE,
    permission_name VARCHAR(60) NOT NULL REFERENCES permissions (name) ON DELETE CASCADE,
    PRIMARY KEY (role_name, permission_name)
);

INSERT INTO role_permissions (role_name, permission_name)
SELECT 'admin', name FROM permissions
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name) VALUES
    ('pengelola', 'module:list'),
    ('pengelola', 'module:manage'),
    ('pengelola', 'batch:list'),
    ('pengelola', 'claim:read:any'),
    ('pengelola', 'claim:verify'),
    ('pengelola', 'progress:read:any'),
    ('pengelola', 'certificate:issue'),
    ('pengelola', 'certificate:verify'),
    ('pengelola', 'content:manage'),
    ('peserta', 'module:list'),
    ('peserta', 'batch:list'),
    ('peserta', 'claim:create'),
    ('peserta', 'claim:read:own'),
    ('peserta', 'claim:cancel'),
    ('peserta', 'progress:read:own'),
    ('peserta', 'progress:write:own'),
    ('peserta', 'certificate:read:own'),
    ('peserta', 'certificate:verify')
ON CONFLICT DO NOTHING;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_fkey;

ALTER TABLE users
    ADD CONSTRAINT users_role_fkey
    FOREIGN KEY (role) REFERENCES roles (name) ON UPDATE CASCADE;
