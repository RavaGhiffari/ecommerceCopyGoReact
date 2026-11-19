# DEV_GUIDE.md — cara bekerja di repo ini

Panduan untuk developer yang **sedang belajar Go** dan terbiasa bekerja dengan instance lokal (tanpa container). Panduan ini menjelaskan *bagaimana berpikir* saat membuat fitur, *bagaimana alat-alatnya bekerja* (Makefile, Docker, migrasi, sqlc), dan *bagaimana alurnya di frontend*.

Dokumen lain punya tugas berbeda, jangan menduplikasi isinya di sini:

| Dokumen | Isi |
|---|---|
| [CLAUDE.md](../CLAUDE.md) | aturan: arsitektur, aturan domain, konvensi |
| [PROJECT_STATE.md](PROJECT_STATE.md) | keadaan terverifikasi, keputusan beserta alasannya, bug yang pernah terjadi |
| [PLAN.md](../PLAN.md) | rencana fase |
| **DEV_GUIDE.md** (ini) | cara kerja sehari-hari dan cara belajar |

Perintah di sini dijalankan dari **root repo** kecuali disebut lain. Bagian 5 (mode lokal) berisi perintah standar yang **belum dicoba di mesin ini**, karena Go tidak terpasang di host; bagian lainnya sudah dipakai langsung.

---

## 1. Peta mental: satu request dari ujung ke ujung

```
Browser ──► Vite (:5173/5174) ──proxy /api──► Gin API (:8081) ──► PostgreSQL
 React                                          │
 pages → features → services ──fetch──►  router → middleware → handler → service → repository → sqlc → pgxpool
```

Di backend, setiap lapisan punya satu tugas (lihat `backend/internal/`):

| Lapisan | File | Tugas | Tidak boleh |
|---|---|---|---|
| Router | `router/router.go` | daftar semua URL, urutan middleware | berisi logika |
| Handler | `modules/<x>/handler.go` | baca request HTTP, panggil service, tulis respons | menjalankan SQL, memegang aturan bisnis |
| Service | `modules/<x>/service.go` | **aturan bisnis**, batas transaksi | mengimpor Gin (`*gin.Context`) |
| Repository | `modules/<x>/repository.go` | membungkus query sqlc, menerjemahkan error database | berisi aturan bisnis |
| Query | `db/queries/*.sql` → `internal/database/queries/*.go` | SQL yang diketik aman | diedit tangan (file `.go`-nya hasil generate) |

Cara membacanya: ikuti satu endpoint dari `router.go` ke handler, lalu service, lalu repository, lalu file `.sql`. Mulai dengan `POST /reviews` (`modules/reviews/`), karena strukturnya lengkap dan ringkas.

---

## 2. Setup pertama dan rutinitas harian

### Pertama kali

```bash
cp .env.example .env          # lalu isi POSTGRES_PASSWORD (tanpa karakter @ : / ? # %)
make up                       # build + jalankan postgres dan api di Docker
make migrate-up               # terapkan semua migrasi ke database dev
make seed-dev                 # data demo (kategori, toko, produk); aman dijalankan ulang
cd frontend && npm install
npm run dev -- --port 5174    # buka http://localhost:5174
```

`.env` berisi kredensial asli dan **tidak pernah** di-commit. Jangan menampilkannya di chat atau log.

Akun demo (hanya lokal): `penjual1@contoh.id` … `penjual4@contoh.id`, kata sandi `penjual-123`. Admin dibuat dengan `make seed-admin email=... name="..."`.

### Port yang dipakai

| Port | Dipakai oleh |
|---|---|
| 5433 | PostgreSQL di Docker (host) |
| 8081 | API Go |
| 5432 | PostgreSQL lokal Anda sendiri, tidak disentuh repo ini |
| 5173 | aplikasi Laravel Anda (IPv4), jadi pakai `--port 5174` untuk Vite |

### Kapan harus melakukan apa

| Anda mengubah… | Lakukan |
|---|---|
| Kode **frontend** | tidak perlu apa-apa, Vite memuat ulang otomatis (HMR) |
| Kode **Go** (backend) | `make up` ulang (atau `docker compose up -d --build api`). API di container **tidak** reload otomatis |
| File **migrasi** baru | `make migrate-up`, lalu `make sqlc` bila ada query yang berubah, lalu rebuild API |
| File **`db/queries/*.sql`** | `make sqlc`, lalu rebuild API |
| `.env` | `make up` ulang (catatan: ganti `POSTGRES_PASSWORD` tidak mengubah password di volume lama, lihat bagian 12) |

### Cek kesehatan cepat

```bash
make ps                                  # postgres harus "healthy"
curl http://127.0.0.1:8081/health        # API hidup
curl http://127.0.0.1:8081/ready         # API + database
make test                                # unit test
make test-integration                    # integration test (butuh `make up`)
cd frontend && npm run build && npm run lint
```

---

## 3. Makefile

`make` adalah **penjalan tugas**. File `Makefile` berisi *target* (nama perintah) dan *recipe* (perintah shell di bawahnya, **harus diawali Tab, bukan spasi**).

```make
fmt:
	$(GO) gofmt -l -w .
```

Hal penting di Makefile ini:

- **Hanya Docker yang dibutuhkan.** Variabel `GO` berisi `docker run ... golang:1.27.4-alpine3.24` (tag pasti sejak 7e, supaya isinya tidak berubah diam-diam), jadi `make test` menjalankan `go test` *di dalam container* dengan folder `backend/` terpasang. Itu sebabnya perintah yang sama jalan walau Go tidak terpasang di Windows.
- **Cache dengan named volume** (`gomodcache`, `gobuildcache`) membuat unduhan modul dan hasil build tidak diulang tiap run.
- **Argumen** diteruskan sebagai variabel: `make migrate-create name=add_wishlist`. Bagian `ifndef name` memberi pesan usage kalau Anda lupa.
- **`MSYS_NO_PATHCONV=1`**: Git Bash di Windows mengubah argumen seperti `-w /src` menjadi path Windows dan merusaknya. Baris itu mematikan perilaku tersebut. Kalau Anda menjalankan `docker run` manual dari Git Bash, `export MSYS_NO_PATHCONV=1` dulu.
- `make` di Windows dipasang dengan `scoop install make`.

Daftar target (`make help` menampilkan ringkasannya):

| Kelompok | Target | Fungsi |
|---|---|---|
| Stack | `up` | `docker compose up -d --build` (build ulang image, jalankan di latar belakang) |
| | `down` | hentikan container. **Volume database tetap aman.** Jangan pernah `down -v` kecuali sengaja menghapus data |
| | `logs`, `ps` | log dan status container |
| Migrasi | `migrate-up` | terapkan semua migrasi yang belum |
| | `migrate-down` | batalkan **satu** migrasi |
| | `migrate-version` | lihat nomor versi database sekarang |
| | `migrate-create name=...` | buat sepasang file migrasi kosong |
| | `migrate-force version=N` | hanya untuk database "dirty" (lihat bagian 7) |
| Kode | `sqlc` | generate kode Go dari `db/queries` |
| | `fmt`, `vet` | format dan analisis statis. Jalankan `fmt` terakhir sebelum selesai |
| | `test` | unit test, tanpa database |
| | `test-db` | buat dan migrasi database `marketplace_test` |
| | `test-integration` | integration test (menjalankan `test-db` dulu) |
| Data | `seed-admin`, `seed-dev` | akun admin, data demo |

---

## 4. Docker dan Compose

### Konsep dasar

- **Image** = cetakan yang tidak berubah (mis. `postgres:18.7-trixie`). Tag seperti `postgres:18` "mengambang": isinya bisa diganti penerbitnya, jadi repo ini memakai tag pasti. **Container** = proses yang berjalan dari sebuah image.
- **Volume** = penyimpanan yang bertahan walau container dihapus. Data PostgreSQL ada di volume `pgdata`.
- **Compose** (`docker-compose.yml`) = satu file yang mendefinisikan beberapa container dan cara mereka tersambung.
- Container tidak melihat `localhost` Anda. Di dalam jaringan Compose, API menyambung ke database lewat nama service: `postgres:5432`. Dari host, Anda memakai `127.0.0.1:5433` (port yang dipublikasikan).

### Service di repo ini

| Service | Jenis | Catatan |
|---|---|---|
| `postgres` | berjalan terus | `postgres:18.7-trixie`, healthcheck `pg_isready`, volume `pgdata`, host `127.0.0.1:5433` |
| `api` | berjalan terus | dibangun dari `backend/Dockerfile`, menunggu `postgres` sehat, host `127.0.0.1:8081` |
| `migrate` | **sekali jalan** | image `migrate/migrate`, hanya aktif lewat profil `tools`, menjalankan migrasi ke database dev |
| `migrate-test` | sekali jalan | sama, tetapi ke database `marketplace_test` |

Port dipublikasikan hanya ke `127.0.0.1` (loopback), jadi database dan API tidak terlihat dari jaringan lokal.

### Dockerfile backend (multi-stage)

1. **Stage `build`**: image Go, `go mod download` dulu (supaya layer ini ter-cache selama `go.mod` tidak berubah), lalu `go build ./cmd/...` menghasilkan *semua* binary: `api`, `seed-admin`, `seed-dev`.
2. **Stage runtime**: `alpine` kecil, pengguna non-root `app`, hanya menyalin binary hasil build. Image akhirnya kecil dan tidak membawa compiler.

### Perintah Docker yang berguna

```bash
docker compose ps                         # = make ps
docker compose logs -f api                # log API saja
docker compose exec postgres sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'   # masuk ke psql
docker compose up -d --build api          # build ulang dan ganti hanya API
```

Catatan: nama database dev di mesin Anda mengikuti `POSTGRES_DB` di `.env` (di mesin ini bernama `main`). Karena itu perintah psql di atas memakai variabel dari dalam container, bukan nama yang di-hardcode.

---

## 5. Bekerja dengan instance lokal (tanpa containerized env)

Anda boleh menjalankan backend langsung di mesin. Ada dua tingkat. **Tingkat A (hibrida) lebih mudah dan saya sarankan.**

> Bagian ini belum dijalankan di mesin pengembangan repo ini (Go belum terpasang di sana). Perintahnya standar, tetapi kalau ada yang tidak cocok, catat dan perbaiki dokumen ini.

### Prasyarat

- **Go 1.27.3** (versi di `backend/go.mod`; pasang dari go.dev/dl). Cek: `go version`.
- Node.js untuk frontend (sudah Anda punya).

### Tingkat A — hanya PostgreSQL di Docker, API dijalankan lokal

```powershell
docker compose up -d postgres            # hanya database, bukan api
docker compose stop api                  # kalau container api masih jalan: bebaskan port 8081
make migrate-up                          # migrasi tetap lewat Makefile (menyambung ke postgres di Docker)

cd backend
$env:DATABASE_URL = "postgres://USER:PASSWORD@127.0.0.1:5433/NAMA_DB?sslmode=disable"
go run ./cmd/api
```

(`USER`, `PASSWORD`, `NAMA_DB` diambil dari `.env` Anda; di Git Bash gunakan `export DATABASE_URL=...`.)

Keuntungannya: mengubah Go tinggal Ctrl+C lalu `go run` lagi, tanpa build image. `make sqlc`, `make migrate-*` dan `make test-integration` tetap bekerja seperti biasa.

Test langsung di mesin (tanpa Docker untuk Go-nya):

```powershell
cd backend
go test ./...                                              # unit test
$env:TEST_DATABASE_URL = "postgres://USER:PASSWORD@127.0.0.1:5433/marketplace_test?sslmode=disable"
go test -tags=integration -count=1 -p 1 ./...              # integration (jalankan `make test-db` dulu)
go test ./internal/modules/reviews -run TestMaskName -v    # satu test saja
```

### Tingkat B — semuanya lokal, termasuk PostgreSQL

Anda sudah punya PostgreSQL 18 lokal di 5432. Buat database di sana (mis. `marketplace_dev`), isi `DATABASE_URL`-nya, lalu:

- **Migrasi**: Makefile mengarah ke database di Docker, jadi di sini Anda memakai CLI `golang-migrate` langsung: `migrate -path backend/db/migrations -database "<URL>" up`.
- **sqlc**: pasang sqlc `1.29.0` (versi yang sama dengan Makefile) atau tetap pakai `make sqlc` (tidak butuh database, hanya membaca file).
- Database test (`marketplace_test`) harus Anda buat dan migrasi sendiri. **Nama database test harus mengandung `_test`**: integration test menolak URL lain karena tes itu menghapus data.

### Variabel environment backend

| Variabel | Default | Catatan |
|---|---|---|
| `DATABASE_URL` | **wajib** | satu-satunya yang wajib |
| `APP_PORT` | `8081` | Vite meneruskan `/api` ke port ini |
| `APP_ENV` | `development` | `production` mengaktifkan cookie `Secure` dan menolak `seed-dev` |
| `LOG_LEVEL` | `info` | |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:5173` | |
| `SESSION_TTL` | `24h` | |
| lainnya | lihat `internal/config/config.go` | timeout, Argon2, pool koneksi, `CHECKOUT_TTL`, ongkir |

Frontend tidak berubah: `npm run dev -- --port 5174` dan Vite meneruskan `/api` ke `127.0.0.1:8081`. Pastikan **hanya satu** API yang memakai port 8081 (container atau lokal, jangan dua-duanya).

---

## 6. Alur berpikir untuk membuat sebuah fitur

Jangan mulai dari mengetik kode. Urutan yang dipakai di repo ini:

### Langkah 0 — Tulis dulu apa yang akan dibuat
- Data apa yang disimpan? Siapa yang boleh membaca dan mengubahnya? (ownership: aturan di CLAUDE.md §6)
- Apa yang terjadi bila dua request datang bersamaan? Apa yang terjadi bila gagal di tengah?
- Tulis jawabannya di `PLAN.md` (pertanyaan terbuka + usulan default), baru kerjakan setelah disetujui.

### Langkah 1 — Dari bawah ke atas, satu potongan kecil

| # | Lapisan | Dikerjakan di | Verifikasi |
|---|---|---|---|
| 1 | Skema database | `backend/db/migrations/NNNNNN_*.sql` | `migrate-up → down → up` |
| 2 | Query SQL | `backend/db/queries/<modul>.sql` | `make sqlc` (gagal bila SQL salah) |
| 3 | Model + aturan | `modules/<x>/model.go`, `service.go` | unit test |
| 4 | Repository | `modules/<x>/repository.go` | dipakai service |
| 5 | DTO + handler | `modules/<x>/dto.go`, `handler.go` | |
| 6 | Route | `router/router.go`, `cmd/api/main.go` | `curl` |
| 7 | Integration test | `router/<x>_integration_test.go` | `make test-integration` |
| 8 | Tipe + service frontend | `frontend/src/types`, `src/services` | `npm run build` |
| 9 | UI | `src/features`, `src/pages` | browser |

### Langkah 2 — Verifikasi sebelum bilang selesai
`make fmt vet test test-integration`, `npm run build && npm run lint`, lalu coba di browser. **"Bisa di-compile" bukan "sudah selesai".**

### Contoh nyata: fitur ulasan (Phase 6a dan 6b)
Semua langkah di atas ada di commit-nya: `000012_create_reviews`, `db/queries/reviews.sql`, `modules/reviews/` (model, repository, service, dto, handler), route di `router.go`, `router/reviews_integration_test.go`, lalu `types/reviews.ts`, `services/reviews.ts`, `features/reviews/`. Membaca file-file itu berurutan adalah latihan yang baik.

Tiga pertanyaan yang dijawab fitur itu dan layak ditiru:
1. **Siapa pemilik data?** `reviews.user_id` disalin dari pesanan, bukan dari request; query item difilter `buyer_id = pemanggil`, jadi item orang lain terlihat seperti "tidak ada" (404).
2. **Bagaimana kalau dua request bersamaan?** Counter rating diubah dengan `UPDATE … SET x = x + delta` (atomik). Test balapan membuktikan: tanpa itu, 20 ulasan serempak hanya tercatat 6.
3. **Apa yang menjaga data benar walau kode salah?** Constraint database: `CHECK rating 1..5`, `UNIQUE (order_item_id)`.

---

## 7. Migrasi database

Migrasi = file SQL bernomor yang mengubah skema **selangkah demi selangkah**. Tiap perubahan punya `up` (terapkan) dan `down` (batalkan). Alatnya `golang-migrate`; Makefile membungkusnya.

### Membuat migrasi

```bash
make migrate-create name=create_wishlist_items
```

Perintah itu membuat dua file kosong dengan nomor berikutnya (6 digit, berurutan):

```
backend/db/migrations/000013_create_wishlist_items.up.sql
backend/db/migrations/000013_create_wishlist_items.down.sql
```

Isi `up.sql` (contoh, hanya ilustrasi):

```sql
CREATE TABLE wishlist_items (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    product_id UUID        NOT NULL REFERENCES products (id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Aturan "satu produk sekali per pengguna" dijaga DATABASE, bukan hanya kode.
    CONSTRAINT wishlist_items_user_product_key UNIQUE (user_id, product_id)
);

-- Indeks mengikuti query nyata: "wishlist saya, terbaru dulu".
CREATE INDEX wishlist_items_user_created_idx ON wishlist_items (user_id, created_at DESC);
```

Isi `down.sql` harus membatalkan persis apa yang dibuat `up`:

```sql
DROP TABLE IF EXISTS wishlist_items;
```

### Menerapkan dan menguji

```bash
make migrate-up         # terapkan
make migrate-version    # harus menunjukkan 13
make migrate-down       # batalkan satu langkah (uji down-nya!)
make migrate-up         # terapkan lagi
make test-db            # migrasi yang sama juga diterapkan ke marketplace_test
```

Kebiasaan wajib: **uji `up → down → up`** sebelum menganggap migrasi benar. Test integration memakai database `marketplace_test` yang dimigrasi oleh `make test-integration`.

### Aturan dari CLAUDE.md §8

- **Jangan pernah mengedit migrasi yang sudah diterapkan.** Buat migrasi baru. (Di repo ini migrasi 1–12 sudah dipakai; mengeditnya membuat database orang lain tidak sinkron.)
- Selalu tulis `down` yang benar.
- Uang = `NUMERIC`, **bukan float**. Waktu = `TIMESTAMPTZ`. Primary key = `UUID`.
- Beri constraint (`NOT NULL`, `CHECK`, `UNIQUE`, `FOREIGN KEY`) untuk semua aturan yang bisa dijaga database. Constraint adalah jaring pengaman terakhir bila kode ada yang lolos.
- Tambahkan indeks hanya untuk query nyata. Jangan menebak.
- Hati-hati dengan perubahan destruktif (hapus kolom, ubah tipe) pada tabel berisi data: pikirkan datanya dulu.

### Kalau migrasi gagal di tengah ("dirty")
Bila satu file SQL gagal sebagian, `migrate-version` menampilkan `dirty`. Perbaiki SQL-nya, rapikan skema secara manual, lalu tandai bersih dengan `make migrate-force version=N`. Perintah ini **hanya** untuk keadaan dirty. Jangan dipakai untuk "melompati" migrasi.

---

## 8. sqlc: dari SQL ke fungsi Go

Anda menulis SQL biasa di `backend/db/queries/<modul>.sql`; sqlc menghasilkan fungsi Go yang tipenya aman. Contoh dari `reviews.sql`:

```sql
-- name: InsertReview :one
INSERT INTO reviews (order_item_id, product_id, store_id, user_id, rating, comment)
VALUES (@order_item_id, @product_id, @store_id, @user_id, @rating, @comment)
ON CONFLICT (order_item_id) DO NOTHING
RETURNING id, order_item_id, product_id, store_id, user_id, rating, comment, status, created_at, updated_at;
```

- `-- name: NamaFungsi :one` → fungsi `InsertReview` yang mengembalikan **satu baris**. `:many` → slice, `:exec` → hanya error.
- `@nama` → parameter bernama; `sqlc.narg('x')` → parameter yang boleh NULL (dipakai untuk filter opsional).
- Komentar tepat di bawah `-- name:` menjadi doc comment fungsi Go.

Setelah mengubah SQL atau migrasi:

```bash
make sqlc
```

Hasilnya ada di `backend/internal/database/queries/*.go`. **Jangan diedit tangan**, dan file itu di-commit supaya build tidak butuh sqlc. Buka file hasil generate setiap kali Anda menambah query. Itu cara tercepat memahami nama struct dan tipenya.

Jebakan yang pernah terjadi (detail di PROJECT_STATE §11):
- Parameter yang dipakai dua kali perlu cast, mis. `@id::uuid`.
- Kolom dari `LEFT JOIN LATERAL` dibaca sqlc sebagai non-null. Bungkus dengan `COALESCE(...)::text`.
- Tipe `uuid`, `timestamptz` dan `numeric` sudah dipetakan ke `uuid.UUID`, `time.Time` dan `decimal.Decimal` lewat `sqlc.yaml`.

---

## 9. Go: yang cukup untuk membaca dan menulis di repo ini

Anda tidak perlu menguasai semuanya. Bagian ini adalah peta untuk hal-hal yang paling sering muncul. Ketik ulang contoh kecil di [go.dev/tour](https://go.dev/tour) bila ada yang belum jelas.

### Struktur file dan paket
```go
package reviews            // semua file di satu folder berbagi satu package

import (
    "context"              // paket standar
    "github.com/google/uuid"   // paket pihak ketiga (tercatat di go.mod)
)
```
Nama yang diawali **huruf kapital** (`Create`, `Review`) bisa dipakai paket lain (diekspor); huruf kecil (`paging`, `toReview`) hanya di paket itu.

### Variabel dan tipe
```go
count := 0                 // := deklarasi + isi (tipe ditebak)
count = 5                  // = mengisi variabel yang sudah ada
var ids []uuid.UUID        // slice (array dinamis)
byID := map[string]int{}   // map
```

### Struct dan method
```go
type Service struct {      // seperti class tanpa pewarisan
    pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {        // "konstruktor" = fungsi biasa
    return &Service{pool: pool}
}

func (s *Service) Create(...) (Review, error) { ... } // (s *Service) = receiver, mirip `this`
```
`*` berarti **pointer** (referensi). `&x` mengambil pointer dari `x`. Pointer juga dipakai agar nilai bisa `nil` (tidak ada), mis. `Review *ItemReview`.

### Error adalah nilai biasa
Go tidak punya `try/catch`. Fungsi yang bisa gagal mengembalikan `error` sebagai nilai terakhir, dan pemanggil **wajib memeriksanya**:

```go
item, err := repo.ReviewableItem(ctx, userID, orderItemID)
if err != nil {
    return err                      // naikkan ke atas
}
```
Pola yang sering dipakai:
```go
return fmt.Errorf("insert review: %w", err)   // bungkus error dengan konteks
errors.Is(err, ErrNotFound)                   // cek jenis error (sentinel)
errors.As(err, &fields)                       // cek error bertipe tertentu
```
Handler yang memetakan error ke status HTTP: lihat `fail()` di `modules/reviews/handler.go`.

### `context.Context`
Parameter pertama hampir semua fungsi yang menyentuh database atau jaringan. Membawa batas waktu dan pembatalan request. Anda cukup meneruskannya (`ctx`) ke fungsi berikutnya.

### `defer`
Menunda sebuah perintah sampai fungsi selesai, terutama untuk bersih-bersih: `defer pool.Close()`. Beberapa `defer` berjalan **terbalik** (terakhir didaftarkan, pertama dijalankan). Komentar di `cmd/api/main.go` menjelaskan urutannya.

### Transaksi di repo ini
Service-lah yang menentukan batas transaksi dengan `database.WithTx`. Jika fungsi di dalamnya mengembalikan error, semua perubahan di-**rollback**:

```go
err = database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
    repo := NewRepository(tx)          // repository yang sama, tapi memakai tx
    item, err := repo.ReviewableItem(ctx, userID, orderItemID)
    if err != nil { return err }
    ...
    return repo.AdjustProductRating(ctx, item.ProductID, 1, in.Rating)
})
```

### Hal lain yang akan Anda temui
- **Generics**: `Page[T any]` di `reviews/model.go`, satu tipe halaman untuk beberapa jenis data.
- **Goroutine dan channel**: `go func() { ... }()` menjalankan sesuatu secara paralel (sweeper kedaluwarsa checkout di `main.go`). Anda jarang perlu menulisnya.
- **Struct tag**: `` `json:"orderItemId"` `` menentukan nama field di JSON. Ini kontrak dengan frontend.
- **Test**: file `*_test.go`, fungsi `TestXxx(t *testing.T)`, pustaka `testify` (`assert`, `require`). Test dengan `//go:build integration` di baris pertama hanya jalan dengan `-tags=integration`.
- **`gofmt`**: format kode tunggal, tidak ada perdebatan gaya. `make fmt` merapikan semuanya.

### Cara membaca satu file Go
1. Lihat `import` dan tipe di atas (struct, konstanta, error).
2. Baca tanda tangan fungsi: nama, parameter, hasil.
3. Baca isi fungsi dari atas ke bawah; setiap `if err != nil` adalah satu kemungkinan gagal.

---

## 10. Pola backend yang perlu Anda ikuti

- **Handler** hanya: ambil parameter, `request.BindJSON` (JSON ketat: hanya `application/json`, field tak dikenal ditolak, batas 1 MiB), panggil service, tulis respons dengan `response.OK` / `response.Fail`.
- **Service** memegang aturan, validasi (`validation.Errors` menjadi 422 dengan detail per field), dan transaksi. Service mengembalikan error biasa; handler yang menerjemahkannya ke HTTP.
- **Ownership**: ID pemilik selalu dari sesi (`authctx`), tidak pernah dari request. Query difilter pemilik; data milik orang lain = 404.
- **Race condition**: bila "baca lalu tulis" bisa terjadi serempak, gunakan `SELECT … FOR UPDATE` atau `UPDATE … SET x = x + 1`. Buktikan dengan test balapan, dan **cek bahwa test itu gagal bila kuncinya dicopot**. Kebiasaan ini membedakan test yang benar-benar menjaga sesuatu.
- **Respons seragam**: `{success, data, message}` atau `{success:false, error:{code, message, details}}`. Jangan membocorkan error SQL ke klien.
- **Uang**: string desimal `"349000.00"`, `decimal.Decimal` di Go, **tidak pernah float**.

---

## 11. Alur frontend

```
src/types/        bentuk data; salinan manual dari DTO Go (harus sinkron!)
src/services/     SATU-SATUNYA tempat memanggil API
src/features/     komponen + hook per fitur (reviews/, orders/, cart/, ...)
src/pages/        halaman, didaftarkan di src/app/router.tsx
src/components/   ui/ (shadcn), shared/, layout/, product/
```

### Menambah fitur di frontend
1. **Tipe** di `src/types/<x>.ts`, cocokkan persis dengan `dto.go`. Bila backend berubah, ubah keduanya sekaligus.
2. **Service** di `src/services/<x>.ts` (fungsi tipis di atas `apiRequest`).
3. **Komponen** di `src/features/<x>/`. Data dimuat dengan `useAsync(() => fetchSesuatu(), kunciUnik)`.
4. Implementasikan **empat keadaan**: loading, kosong, error, sukses. Untuk error dan kosong sudah ada `ErrorState` / `EmptyState` di `components/shared/StateViews.tsx`.
5. **Form**: React Hook Form + Zod (`schema.ts` di folder fitur), error dari server dipetakan ke field lewat `applyServerError`. Contoh lengkap: `features/reviews/ReviewDialog.tsx`.
6. **Halaman** baru → tambahkan route di `src/app/router.tsx`. Pagar `RequireAuth` / `RequireSeller` hanya soal UX; **keamanan sebenarnya di backend**.

### Komponen UI (shadcn/ui)
Komponen ada di `src/components/ui/`. Menambah komponen baru: `npx shadcn add <nama>`. **Setelah itu wajib** memperbaiki impor yang salah dari CLI:

```bash
sed -i 's#from "cn"#from "@/lib/utils"#' src/components/ui/*.tsx
```
(Atau ubah manual `from "cn"` menjadi `from "@/lib/utils"`; kalau tidak, build gagal.)

### Perintah
```bash
cd frontend
npm run dev -- --port 5174   # dev server
npm run build                # cek tipe (tsc) + build produksi
npm run lint                 # oxlint (bukan ESLint)
```

### Mengapa Vite meneruskan `/api`
Di mode dev browser hanya bicara ke Vite, dan Vite meneruskan `/api` ke API. Dari sudut pandang browser semuanya satu origin, jadi cookie sesi menjadi cookie biasa dan CORS tidak terlibat. Opsi `changeOrigin: false` penting: pemeriksaan CSRF membandingkan header `Origin` dengan `Host`, dan mengubah `Host` membuat setiap POST dianggap lintas situs.

---

## 12. Pengujian dan debugging

### Tiga lapis test

| Lapis | Perintah | Apa yang diuji |
|---|---|---|
| Unit (Go) | `make test` | aturan murni: validasi, hitungan, tabel transisi |
| Integration (Go) | `make test-integration` | HTTP sampai database sungguhan: kepemilikan, constraint, race |
| E2E (browser) | skrip `playwright-core` di `docs/e2e-reference/` | alur pengguna sungguhan |

Menjalankan satu test integration di container (dari Git Bash, setelah `export MSYS_NO_PATHCONV=1`):

```bash
docker run --rm --network goframework_default --env-file .env \
  -v "$PWD/backend:/src" -v gomodcache:/go/pkg/mod -v gobuildcache:/root/.cache/go-build -w /src golang:1.27.4-alpine3.24 \
  sh -c 'TEST_DATABASE_URL="postgres://$POSTGRES_USER:$POSTGRES_PASSWORD@postgres:5432/marketplace_test?sslmode=disable" \
         go test -tags=integration -count=1 -run TestReviews ./internal/router/'
```

Skrip E2E bersifat manual dan folder `docs/e2e-reference/` di-ignore git. Pasang `playwright-core` di folder scratch (bukan di repo), jalankan Vite di port 5190, dan jalankan skripnya dari PowerShell (Edge headless tidak jalan dari sandbox Bash). Cara lengkap ada di `docs/e2e-reference/README.md`. Registrasi dibatasi 10 per jam per IP, jadi jangan menjalankan skrip yang mendaftar akun berkali-kali.

### Memeriksa API dengan curl
Request yang mengubah data harus menyertakan header `Origin` yang sama dengan host (pemeriksaan CSRF):

```bash
B=http://127.0.0.1:8081/api/v1; O="Origin: http://127.0.0.1:8081"
curl -c jar -H "$O" -H "Content-Type: application/json" \
     -d '{"email":"penjual1@contoh.id","password":"penjual-123"}' $B/auth/login
curl -b jar $B/me/reviews/pending
```

### Memeriksa database
```bash
docker compose exec postgres sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
```
Dua invariant yang sebaiknya selalu nol setelah mengubah stok atau rating ada di `PROJECT_STATE.md` §5.

### Masalah yang sudah pernah terjadi

| Gejala | Penyebab | Solusi |
|---|---|---|
| `docker run -w /src` error path aneh di Git Bash | Git Bash mengubah argumen | `export MSYS_NO_PATHCONV=1` |
| API crash-loop `password authentication failed` setelah mengganti `POSTGRES_PASSWORD` | image postgres hanya memakai variabel itu saat volume **kosong** | jalankan `ALTER USER` di dalam container (PROJECT_STATE §1d), lalu restart `api` |
| Build frontend gagal `Cannot find module "cn"` | CLI shadcn menulis impor yang salah | ganti `from "cn"` → `from "@/lib/utils"` |
| `localhost:5173` menampilkan aplikasi lain | port IPv4 dipakai Laravel Anda | `--port 5174` |
| Registrasi 429 `RATE_LIMITED` | batas 10/jam per IP; di dev semua request dari satu IP | `docker compose restart api` |
| Migrasi `dirty` | SQL gagal di tengah | perbaiki lalu `make migrate-force version=N` (bagian 7) |
| Mengubah Go tapi perilaku tidak berubah | container `api` belum di-build ulang | `docker compose up -d --build api` |
| Logout langsung kembali ke `/login` | urutan navigasi dan state | sudah ditangani dengan `flushSync` (PROJECT_STATE §11) |

**Jangan mematikan proses berdasarkan port** sebelum membaca command line-nya; port 5173 dan 5432 milik aplikasi Anda yang lain.

---

## 13. Latihan bertahap

**Tingkat 1: membaca**
1. Ikuti `GET /products/:id/reviews` dari `router.go` sampai SQL-nya, dan tulis dengan kata-kata sendiri apa yang terjadi di tiap lapisan.
2. Buka `modules/reviews/model_test.go` dan ubah satu angka di tabel test sampai test gagal; baca pesan gagalnya.

**Tingkat 2: perubahan kecil dan aman**
1. Tambahkan field baru ke respons `GET /categories` (mis. jumlah produk). Alur: SQL → `make sqlc` → repository → DTO → tipe TypeScript → tampilkan.
2. Tambahkan satu aturan validasi baru dan test unitnya.

**Tingkat 3: fitur kecil lengkap (mis. wishlist)**
1. Tulis rencananya di `PLAN.md` (data, endpoint, kepemilikan, kasus gagal).
2. Migrasi `up/down`, query, modul, route, integration test (termasuk "produk orang lain → 404" dan "tambah dua kali → tidak error ganda").
3. Tipe, service, dan UI frontend, dengan empat keadaan.
4. Jalankan seluruh perintah verifikasi di bagian 2 dan 12, lalu ringkas apa yang Anda pelajari di `PROJECT_STATE.md`.

---

## 14. Aturan emas (ringkasan CLAUDE.md §16)

- Periksa kode yang ada sebelum mengubah apa pun. Jangan menimpa konfigurasi tanpa melihatnya.
- Potongan kecil, satu fitur sekali jalan; berhenti di akhir tiap slice untuk review.
- Jangan commit atau push otomatis, jangan menyentuh `.env`, jangan menaruh rahasia di kode.
- Backend adalah satu-satunya penjaga keamanan; frontend hanya UX.
- Jangan bilang "selesai" sebelum test, build, dan pengecekan di browser dijalankan. Bila ada yang gagal, laporkan apa adanya.
