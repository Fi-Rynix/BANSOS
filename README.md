# BANSOS

Nama sistem: BANSOS (Bagi Akses, Naik Skill, Online Sertifikat)

Tagline: war-nya belajar, hasilnya sertifikat.

## Deskripsi

BANSOS adalah sistem distribusi modul pembelajaran IT bersertifikat dengan kuota terbatas. Penyelenggara membuka gelombang penyaluran (batch) berisi slot belajar yang jumlahnya terbatas, lalu peserta berebut slot tersebut dalam waktu terbatas (war). Yang berhasil mendapat slot menyelesaikan modul pembelajaran, dan setelah lulus penilaian menerima sertifikat digital bernomor unik yang dapat diverifikasi secara publik.

Sistem mencegah dua peserta mendapat slot terakhir yang sama secara bersamaan, mencegah satu peserta mengklaim dua slot pada batch yang sama, dan memastikan satu nomor sertifikat hanya pernah diterbitkan satu kali.

## Role pada Sistem

| Role | Hak akses utama |
|------|-----------------|
| Admin | Mengelola user, katalog modul, dan batch kuota |
| Pengelola | Menyetujui atau menolak klaim, memvalidasi kelulusan, menerbitkan dan mencabut sertifikat |
| Peserta | Melihat katalog, war slot, belajar dan mengerjakan kuis, klaim sertifikat, melihat riwayat |

## Fitur

| No | Fitur | Role |
|----|-------|------|
| 1 | Registrasi dan login | Semua |
| 2 | Kelola modul pembelajaran (tambah, lihat, ubah, hapus) | Admin |
| 3 | Kelola batch kuota per modul (buka, tutup, jadwal war) | Admin |
| 4 | War slot: klaim slot batch yang sedang buka | Peserta |
| 5 | Setujui atau tolak klaim, promosi dari waiting list | Pengelola |
| 6 | Batalkan klaim | Peserta |
| 7 | Belajar: baca konten modul, tandai lesson selesai, kerjakan kuis | Peserta |
| 8 | Klaim sertifikat setelah lulus, pengelola memvalidasi | Peserta, Pengelola |
| 9 | Verifikasi sertifikat publik lewat nomor serial | Semua (tanpa login) |
| 10 | Riwayat klaim dan log perubahan status | Peserta, Pengelola |
| 11 | Notifikasi email hasil war, pengingat deadline, expire batch otomatis | Background |

## Rancangan Database

### PostgreSQL (data transaksional)

| Tabel | Kolom utama | Relasi |
|-------|-------------|--------|
| users | id, name, email (unik), password_hash, role_id (FK ke roles.id), created_at | Satu user punya satu role, banyak klaim dan sertifikat |
| roles | id, name (unik), description | Satu role punya banyak user dan banyak permission |
| permissions | id, code (unik), description | Dipakai banyak role |
| role_permissions | role_id, permission_id | Penghubung role dan permission, unik per pasangan |
| modules | id, code (unik), title, category, level, description, is_active, created_at | Satu modul punya banyak batch |
| batches | id, module_id, quota, allocated, opens_at, closes_at, status, created_at | Milik satu modul, punya banyak klaim |
| claims | id, user_id, batch_id, status, claimed_at, decided_at, created_at | Milik satu user dan satu batch; unik per (user_id, batch_id) |
| claim_logs | id, claim_id, actor_id, from_status, to_status, note, created_at | Mencatat setiap perubahan status klaim |
| certificates | id, serial (unik), user_id, module_id, claim_id, status, issued_at, revoked_at, created_at | Milik satu user dan satu modul; unik per (user_id, module_id) |

### MongoDB (konten dan progres)

| Koleksi | Bentuk dokumen | Keputusan desain |
|---------|----------------|------------------|
| module_contents | module_id, title, chapters (array of chapter, tiap chapter berisi array lesson: slug, title, body) | Konten di-embed karena selalu dibaca utuh per modul dan tidak di-query terpisah |
| progress | user_id, module_id, completed_lessons (array slug), quiz_scores, updated_at | Satu dokumen per user per modul; update atomik dan idempoten |
| module_reviews | module_id, user_id, rating, body, created_at | Opsional (stretch): ulasan modul dengan index gabungan |

### Alur status

| Entitas | Alur |
|---------|------|
| Batch | draft, scheduled, open, closed |
| Klaim | pending, approved, rejected, cancelled, expired, waitlisted |
| Sertifikat | requested, issued, revoked |

Titik rebutan (untuk uji konkurensi): dua request war pada slot terakhir batch, dua submit klaim serentak dari user yang sama, dan dua penerbitan sertifikat dengan serial yang sama. Ketiganya harus menghasilkan tepat satu keberhasilan dan sisanya konflik 409.

## ERD

ERD digambar dengan dbdiagram.io. Salin kode berikut ke dbdiagram.io untuk menghasilkan gambar, lalu simpan sebagai erd.png di repo.

```
Table users {
  id serial [pk]
  name varchar
  email varchar [unique]
  password_hash varchar
  role_id int [ref: > roles.id]
  created_at timestamp
}

Table roles {
  id serial [pk]
  name varchar [unique]
  description varchar
}

Table permissions {
  id serial [pk]
  code varchar [unique]
  description varchar
}

Table role_permissions {
  role_id int [ref: > roles.id]
  permission_id int [ref: > permissions.id]
}

Table modules {
  id serial [pk]
  code varchar [unique]
  title varchar
  category varchar
  level varchar
  description text
  is_active boolean
  created_at timestamp
}

Table batches {
  id serial [pk]
  module_id int [ref: > modules.id]
  quota int
  allocated int
  opens_at timestamp
  closes_at timestamp
  status varchar
  created_at timestamp
}

Table claims {
  id serial [pk]
  user_id int [ref: > users.id]
  batch_id int [ref: > batches.id]
  status varchar
  claimed_at timestamp
  decided_at timestamp
  created_at timestamp
}

Table claim_logs {
  id serial [pk]
  claim_id int [ref: > claims.id]
  actor_id int [ref: > users.id]
  from_status varchar
  to_status varchar
  note text
  created_at timestamp
}

Table certificates {
  id serial [pk]
  serial varchar [unique]
  user_id int [ref: > users.id]
  module_id int [ref: > modules.id]
  claim_id int [ref: > claims.id]
  status varchar
  issued_at timestamp
  revoked_at timestamp
  created_at timestamp
}
```

## Pemetaan Fitur ke Materi Modul

| Modul | Diterapkan di |
|-------|---------------|
| 2. REST API dan HTTP | Metode dan status code yang tepat di seluruh endpoint, termasuk 409 saat konflik war |
| 3. Database dan Repository | Migrasi PostgreSQL, query berparameter, filter dan paginasi di sisi database |
| 4. Clean Architecture | Layer model, repository, service, delivery sesuai struktur modul kuliah |
| 5. Authentication | Register, login, hash password, access dan refresh token, rate limit login |
| 6. Authorization dan RBAC | Tabel roles, permissions, role_permissions; ownership di layer service |
| 7. Advanced API Design | Validator deklaratif, cursor pagination katalog dan riwayat, error code stabil plus request id |
| 8. UTS | Integrasi seluruh lapisan menjadi satu alur utuh |
| 9. Transactions dan Concurrency | War slot atomik, unique constraint anti klaim ganda, nomor sertifikat atomik, uji beban serentak |
| 10. MongoDB | Konten modul embedded, progres idempoten, index unik dan gabungan |
| 11. Background Jobs | Outbox, Asynq, email hasil war dan sertifikat, expire batch dan release slot |
| 12. Testing, Swagger, Docker | Unit test rules, integration test war, Swagger UI, docker compose penuh |

## Struktur Folder (Rencana Boilerplate)

Mengikuti struktur Clean Architecture yang dipakai modul praktikum sepanjang semester, tanpa istilah controller atau handler. Struktur pada panduan dosen hanya ilustrasi, dan panduan itu sendiri meminta menyesuaikan nama folder, nama layer, dan pembagian file dengan modul praktikum.

```
bansos-backend/
├── main.go            titik masuk API, graceful shutdown, flag role
├── worker.go          relay outbox dan worker Asynq
├── app/
│   ├── model/         user, module, batch, claim, certificate, DTO request dan response
│   ├── repository/    query PostgreSQL, dbtx, tx manager, koleksi MongoDB
│   └── service/       auth, module, batch, claim, certificate, content, progress, fungsi rules murni
├── config/            app (perakitan Fiber dan ErrorHandler), env, logger, queue
├── database/          postgres (pgxpool), mongo
├── helper/            response, request, errors, jwt, security, authz, cursor, negotiate, mailer, validator
├── middleware/        logger, recover, cors, requireJSON, auth, permission
├── route/             pemasangan endpoint ke service
├── migrations/        file SQL berurutan
├── tests/             integration, loadtest
├── docs/              hasil swag init
├── .env.example
├── .gitignore
├── go.mod
└── README.md
```

Struktur ini sama dengan pola proyek api-students pada semester ini, hanya diisi domain baru: module, batch, claim, certificate.

## Fitur Utama untuk Boilerplate

Satu fitur yang harus berjalan utuh pada boilerplate: kelola modul pembelajaran (CRUD oleh admin), mengikuti pola contoh kelola data ruang pada panduan. Fitur lain cukup berupa kerangka yang nanti dilengkapi.

## Anggota dan Peran

Pembagian kerja berupa vertical slice per domain: tiap anggota mengerjakan domainnya dari entity sampai route. Skeleton infrastruktur (init module, config, helper dasar, docker compose) dikerjakan bersama di awal lewat PR kecil. Nomor urut migration dikoordinasikan di PR pertama.

| Anggota | Peran | Cakupan | Contoh branch |
|---------|-------|---------|---------------|
| Rafi Ihya Azzaky | Git Master, auth dan RBAC | user, register, login, refresh, roles dan permissions, middleware auth dan permission, file jahitan route.go dan main.go, review PR | feat/auth, feat/rbac |
| Daffa Eka Saputra | Modul, batch, dan war | modul, batch kuota, klaim slot, log klaim, fitur utuh kelola modul, uji beban war | feat/claim-war |
| Valentino Chandra | Konten, sertifikat, dan worker | konten MongoDB, progres, kuis, sertifikat dan verifikasi publik, outbox dan notifikasi email | feat/certificate, feat/worker |
