# Panduan & Spesifikasi Lengkap Flow API Promosi Toko (Merchant Promotions)

Dokumen ini menjelaskan arsitektur, siklus hidup (*lifecycle*), kontrak payload permintaan/respons, serta integrasi lengkap sistem **Promosi Toko** baik untuk sisi Merchant (Owner & Business Admin) maupun sisi Customer (Feed, Notifikasi, dan Etalase Toko).

---

## Daftar Isi
1. [Arsitektur & Konsep Utama](#1-arsitektur--konsep-utama)
2. [Diagram Siklus Hidup & Alur End-to-End](#2-diagram-siklus-hidup--alur-end-to-end)
3. [Daftar Endpoint Merchant](#3-daftar-endpoint-merchant)
   - [3.1 Ambil Produk Eligible](#31-ambil-produk-eligible)
   - [3.2 Buat Promosi Baru](#32-buat-promosi-baru)
   - [3.3 Daftar Promosi (Tab Active, Popular, Ended)](#33-daftar-promosi-tab-active-popular-ended)
   - [3.4 Detail Promosi](#34-detail-promosi)
   - [3.5 Perpanjang Masa Promosi](#35-perpanjang-masa-promosi)
   - [3.6 Hentikan Promosi Lebih Awal](#36-hentikan-promosi-lebih-awal)
4. [Daftar Endpoint Customer](#4-daftar-endpoint-customer)
   - [4.1 Tracking Impresi & Klik Promosi](#41-tracking-impresi--klik-promosi)
   - [4.2 Detail Promosi untuk Customer](#42-detail-promosi-untuk-customer)
5. [Integrasi Etalase Virtual "Promosi"](#5-integrasi-etalase-virtual-promosi)
6. [Integrasi Postingan Feed Otomatis](#6-integrasi-postingan-feed-otomatis)
7. [Mekanisme Notifikasi & Deduplikasi](#7-mekanisme-notifikasi--deduplikasi)
8. [Background Scheduler & Worker](#8-background-scheduler--worker)

---

## 1. Arsitektur & Konsep Utama

### 1.1 Status Promosi
Promosi memiliki 4 status kemungkinan:
- `scheduled`: Tanggal mulai (`start_at`) berada di masa mendatang (akan diaktifkan otomatis oleh worker saat waktunya tiba).
- `active`: Promosi sedang berjalan aktif saat ini (`start_at <= NOW <= end_at`). Diskon produk diterapkan langsung ke katalog.
- `ended`: Promosi selesai secara alami karena melewati batas waktu `end_at`. Diskon otomatis di-rollback ke kondisi sebelum promo.
- `stopped`: Dihentikan lebih awal secara manual oleh merchant. Diskon otomatis di-rollback seketika.

### 1.2 Penanganan Diskon Sebelumnya (Rollback vs Keep)
Saat merchant memilih produk yang sebelumnya sudah memiliki diskon di katalog:
1. **Opsi `keep_existing_discount: true`**: Nilai diskon produk saat ini dipertahankan, namun masa berlaku diskon (`discount_end_at`) diperpanjang mengikuti `end_at` promosi.
2. **Opsi Ubah Diskon Baru (`keep_existing_discount: false`)**: Nilai diskon lama (tipe, nominal/persen, batas tanggal) dicadangkan ke tabel `promotion_products`. Diskon produk di katalog ditimpa dengan diskon promosi. Saat promosi berakhir atau dihentikan, diskon lama akan dipulihkan secara otomatis.

### 1.3 Penguncian Diskon Produk
Selama produk terdaftar dalam promosi yang berstatus `scheduled` atau `active`, pengubahan diskon secara manual melalui endpoint katalog `PUT /business/:business_id/products/:product_id` akan **diblokir** oleh sistem untuk mencegah inkonsistensi data.

---

## 2. Diagram Siklus Hidup & Alur End-to-End

```mermaid
sequenceDiagram
    autonumber
    actor Merchant
    participant API as Backend API
    participant DB as PostgreSQL
    participant Worker as Background Worker
    participant Notif as Push Notification Service
    actor Customer

    Note over Merchant, API: 1. Pembuatan Promosi
    Merchant->>API: GET /business/:id/promotions/eligible-products
    API-->>Merchant: List produk yang bisa dipromosikan (status diskon saat ini)
    Merchant->>API: POST /business/:id/promotions (Caption, Produk, Diskon, Jadwal)
    API->>DB: Simpan Promosi, Cadangkan Diskon Lama, Update Diskon Katalog
    API->>DB: Buat Postingan Otomatis (Badge Promosi)
    
    alt Jika Langsung Aktif (StartAt <= NOW)
        API->>Notif: Kirim notifikasi (Bookmarks Toko + Wishlists Produk)
        Notif-->>Customer: Push Notif & In-App (Foto Produk Utama + Caption)
    else Jika Terjadwal (StartAt > NOW)
        Note over Worker: Worker per menit memeriksa start_at
        Worker->>DB: Ubah status menjadi 'active' saat start_at tercapai
        Worker->>Notif: Picu notifikasi ke Customer saat aktif
    end
    API-->>Merchant: Status Promosi Berhasil Dibuat

    Note over Customer, API: 2. Pengalaman Customer
    Customer->>API: GET /customer/businesses/:id/product-groups
    API-->>Customer: Tampilkan grup virtual "Promosi" di posisi teratas
    Customer->>API: GET /customer/businesses/:id/products?group_id=promotion
    API-->>Customer: Daftar semua produk yang sedang diskon promosi
    Customer->>API: POST /customer/promotions/events (Impression / Click)
    API->>DB: Catat event & perbarui click_count (Dedup Redis 30 Menit)

    Note over Worker, DB: 3. Akhir Promosi / Rollback
    Worker->>DB: Cek promosi yang end_at < NOW
    Worker->>DB: Set status 'ended' & Rollback diskon produk ke nilai awal
```

---

## 3. Daftar Endpoint Merchant

Semua endpoint merchant berada di bawah scope otorisasi bisnis:
- **Base URL:** `/business/:business_id/promotions` (atau alias `/merchant/businesses/:business_id/promotions`)
- **Headers:** `Authorization: Bearer <token>`
- **Izin Admin:** Memerlukan izin `manage_promotions` bagi staf `business_admin`.

---

### 3.1 Ambil Produk Eligible
Mengambil daftar semua produk aktif milik toko yang **belum terikat** dalam promosi aktif atau terjadwal lain.

- **Method:** `GET`
- **URL:** `/business/:business_id/promotions/eligible-products`
- **Response Success (200 OK):**
```json
{
  "responseCode": "00",
  "message": "Daftar produk eligible berhasil diambil",
  "data": [
    {
      "product_id": "PRD-abc12345",
      "product_name": "Kopi Susu Gula Aren",
      "image_url": "https://s3.../preview.jpg",
      "price": 25000,
      "is_eligible": true,
      "has_existing_discount": true,
      "current_discount_type": "percentage",
      "current_discount": 10,
      "current_max_discount": null,
      "current_discount_start_at": "2026-10-01T00:00:00Z",
      "current_discount_end_at": "2026-10-25T23:59:59Z"
    },
    {
      "product_id": "PRD-xyz98765",
      "product_name": "Roti Bakar Cokelat Keju",
      "image_url": "https://s3.../roti.jpg",
      "price": 20000,
      "is_eligible": true,
      "has_existing_discount": false,
      "current_discount_type": "",
      "current_discount": 0,
      "current_max_discount": null,
      "current_discount_start_at": null,
      "current_discount_end_at": null
    }
  ]
}
```

---

### 3.2 Buat Promosi Baru
Membuat promosi toko, menetapkan produk promosi, memilih produk utama untuk preview notifikasi/postingan, dan mengatur diskon masing-masing produk.

- **Method:** `POST`
- **URL:** `/business/:business_id/promotions`
- **Middleware:** `RequireAuth`, `RequireKYC`, `BusinessAccessGuard`
- **Payload Request:**
```json
{
  "caption": "Promo Gajian Spesial! Nikmati diskon hingga 30% untuk menu favorit Anda.",
  "main_product_id": "PRD-abc12345",
  "start_at": "2026-10-20T08:00:00+07:00",
  "end_at": "2026-10-27T23:59:59+07:00",
  "products": [
    {
      "product_id": "PRD-abc12345",
      "keep_existing_discount": true
    },
    {
      "product_id": "PRD-xyz98765",
      "keep_existing_discount": false,
      "discount_type": "percentage",
      "discount": 20,
      "max_discount": 10000
    }
  ]
}
```

#### Aturan Validasi:
1. `caption`: Wajib diisi, minimal 3 karakter, **maksimal 150 karakter**.
2. `main_product_id`: Wajib dipilih dan **harus termasuk** dalam daftar `products`.
3. `start_at` & `end_at`: Rentang promosi tidak boleh melebihi **90 hari**.
4. `products`: Minimal 1 produk, **maksimal 20 produk**.
5. Diskon:
   - Jika `keep_existing_discount: true`, sistem menggunakan nilai diskon yang sudah ada di produk tersebut.
   - Jika `keep_existing_discount: false`, `discount_type` ("percentage" atau "nominal") dan `discount` wajib diisi valid.

- **Response Success (201 Created):**
```json
{
  "responseCode": "00",
  "message": "Promosi berhasil dibuat",
  "data": {
    "promotion_id": "PRM-1a2b3c4d",
    "business_id": "BIZ-001",
    "caption": "Promo Gajian Spesial! Nikmati diskon hingga 30% untuk menu favorit Anda.",
    "status": "scheduled",
    "start_at": "2026-10-20T08:00:00+07:00",
    "end_at": "2026-10-27T23:59:59+07:00",
    "days_remaining": 7,
    "main_product": {
      "product_id": "PRD-abc12345",
      "product_name": "Kopi Susu Gula Aren",
      "image_url": "https://s3.../preview.jpg"
    },
    "products_count": 2,
    "impression_count": 0,
    "click_count": 0,
    "created_at": "2026-10-19T18:00:00+07:00"
  }
}
```

---

### 3.3 Daftar Promosi (Tab Active, Popular, Ended)
Menampilkan daftar promosi milik toko lengkap dengan ringkasan statistik (metrik).

- **Method:** `GET`
- **URL:** `/business/:business_id/promotions?tab=active`
- **Query Params:**
  - `tab`: Pilihan tab:
    - `active` (Default): Menampilkan promosi yang sedang aktif atau terjadwal.
    - `popular`: Menampilkan promosi aktif yang diurutkan berdasarkan `click_count DESC` (Paling Populer).
    - `ended`: Menampilkan promosi yang sudah selesai atau dihentikan manual.
  - `limit`: Jumlah per halaman (default: 20).
  - `page`: Nomor halaman (default: 1).

- **Response Success (200 OK):**
```json
{
  "responseCode": "00",
  "message": "Daftar promosi berhasil diambil",
  "data": {
    "summary": {
      "active_count": 3,
      "total_impressions": 1250,
      "total_clicks": 340
    },
    "promotions": [
      {
        "promotion_id": "PRM-1a2b3c4d",
        "caption": "Promo Gajian Spesial!",
        "status": "active",
        "start_at": "2026-10-18T00:00:00Z",
        "end_at": "2026-10-25T23:59:59Z",
        "days_remaining": 6,
        "main_product_name": "Kopi Susu Gula Aren",
        "main_product_image": "https://s3.../preview.jpg",
        "original_price": 25000,
        "final_price": 22500,
        "products_count": 2,
        "impression_count": 420,
        "click_count": 95,
        "created_at": "2026-10-18T00:00:00Z"
      }
    ]
  }
}
```

---

### 3.4 Detail Promosi
Melihat rincian sebuah promosi beserta seluruh daftar produk, harga awal, harga diskon, dan status pencadangan.

- **Method:** `GET`
- **URL:** `/business/:business_id/promotions/:promotion_id`
- **Response Success (200 OK):**
```json
{
  "responseCode": "00",
  "message": "Detail promosi berhasil diambil",
  "data": {
    "promotion_id": "PRM-1a2b3c4d",
    "business_id": "BIZ-001",
    "caption": "Promo Gajian Spesial!",
    "status": "active",
    "start_at": "2026-10-18T00:00:00Z",
    "end_at": "2026-10-25T23:59:59Z",
    "days_remaining": 6,
    "main_product_id": "PRD-abc12345",
    "main_product": {
      "product_id": "PRD-abc12345",
      "product_name": "Kopi Susu Gula Aren",
      "image_url": "https://s3.../preview.jpg"
    },
    "impression_count": 420,
    "click_count": 95,
    "post_id": "PST-999888",
    "products": [
      {
        "product_id": "PRD-abc12345",
        "product_name": "Kopi Susu Gula Aren",
        "image_url": "https://s3.../preview.jpg",
        "original_price": 25000,
        "final_price": 22500,
        "discount_type": "percentage",
        "discount": 10,
        "max_discount": null,
        "discount_source": "existing"
      },
      {
        "product_id": "PRD-xyz98765",
        "product_name": "Roti Bakar Cokelat Keju",
        "image_url": "https://s3.../roti.jpg",
        "original_price": 20000,
        "final_price": 16000,
        "discount_type": "percentage",
        "discount": 20,
        "max_discount": 10000,
        "discount_source": "promotion"
      }
    ],
    "created_at": "2026-10-18T00:00:00Z",
    "updated_at": "2026-10-18T00:00:00Z"
  }
}
```

---

### 3.5 Perpanjang Masa Promosi
Memperpanjang masa aktif promosi yang sedang `active` atau `scheduled`. Otomatis memperbarui `discount_end_at` pada produk-produk terkait.

- **Method:** `PATCH`
- **URL:** `/business/:business_id/promotions/:promotion_id/extend`
- **Middleware:** `RequireKYC`
- **Payload Request (Pilih salah satu format):**
  - **Opsi A (Tambah hari):**
    ```json
    {
      "add_days": 3
    }
    ```
  - **Opsi B (Tentukan tanggal akhir baru):**
    ```json
    {
      "end_at": "2026-10-30T23:59:59+07:00"
    }
    ```
- **Response Success (200 OK):**
```json
{
  "responseCode": "00",
  "message": "Masa berlaku promosi berhasil diperpanjang",
  "data": {
    "promotion_id": "PRM-1a2b3c4d",
    "end_at": "2026-10-28T23:59:59Z",
    "days_remaining": 9
  }
}
```

---

### 3.6 Hentikan Promosi Lebih Awal
Menghentikan promosi secara instan sebelum tanggal berakhir. Sistem secara otomatis melakukan **rollback** diskon pada produk:
- Produk yang memiliki diskon lama akan dipulihkan ke nilai diskon aslinya.
- Produk yang awalnya tidak memiliki diskon akan dihapus diskonnya dari katalog.

- **Method:** `PATCH`
- **URL:** `/business/:business_id/promotions/:promotion_id/stop`
- **Middleware:** `RequireKYC`
- **Payload:** Kosong `{}`
- **Response Success (200 OK):**
```json
{
  "responseCode": "00",
  "message": "Promosi berhasil dihentikan dan diskon produk telah dipulihkan",
  "data": {
    "promotion_id": "PRM-1a2b3c4d",
    "status": "stopped"
  }
}
```

---

## 4. Daftar Endpoint Customer

### 4.1 Tracking Impresi & Klik Promosi
Mencatat analitik saat promosi dilihat (*impression*) di feed/timeline/profil atau saat diklik (*click*). Dilengkapi **Deduplikasi 30 Menit** berbasis Redis (atau fallback cache memori) per user/IP agar metrik tidak membengkak karena spam atau scroll bolak-balik.

- **Method:** `POST`
- **URL:** `/customer/promotions/events`
- **Middleware:** `OptionalAuth` (Bisa diakses baik login maupun guest)
- **Payload Request:**
```json
{
  "events": [
    {
      "promotion_id": "PRM-1a2b3c4d",
      "product_id": "PRD-abc12345",
      "event_type": "impression",
      "source": "feed"
    },
    {
      "promotion_id": "PRM-1a2b3c4d",
      "product_id": "PRD-abc12345",
      "event_type": "click",
      "source": "post"
    }
  ]
}
```

- **Nilai `event_type`:** `"impression"` atau `"click"`.
- **Nilai `source`:** `"feed"`, `"post"`, `"business_profile"`, `"notification"`, `"product_group"`.
- **Response Success (200 OK):**
```json
{
  "responseCode": "00",
  "message": "Event promosi berhasil dicatat",
  "data": {
    "recorded_count": 2
  }
}
```

---

### 4.2 Detail Promosi untuk Customer
Mengambil rincian promosi untuk tampilan customer saat mengklik kartu promosi atau notifikasi.

- **Method:** `GET`
- **URL:** `/customer/promotions/:promotion_id`
- **Middleware:** `OptionalAuth`
- **Response Success (200 OK):**
```json
{
  "responseCode": "00",
  "message": "Detail promosi berhasil diambil",
  "data": {
    "promotion_id": "PRM-1a2b3c4d",
    "business_id": "BIZ-001",
    "caption": "Promo Gajian Spesial!",
    "status": "active",
    "start_at": "2026-10-18T00:00:00Z",
    "end_at": "2026-10-25T23:59:59Z",
    "days_remaining": 6,
    "main_product": {
      "product_id": "PRD-abc12345",
      "product_name": "Kopi Susu Gula Aren",
      "image_url": "https://s3.../preview.jpg"
    },
    "products": [ ... ]
  }
}
```

---

## 5. Integrasi Etalase Virtual "Promosi"

### 5.1 Penempatan Otomatis di Paling Atas
Grup etalase **"Promosi"** bersifat *virtual* (tidak dibuat sebagai baris fisik permanen di database tabel `product_groups` agar tidak menghapus kategori asli masing-masing produk).

1. Ketika frontend memanggil:
   - Customer: `GET /customer/businesses/:business_id/product-groups`
   - Merchant: `GET /business/:business_id/product-groups`
2. Jika toko memiliki minimal 1 promosi aktif, backend otomatis menyisipkan objek grup virtual di indeks `0`:
   ```json
   {
     "product_group_id": "promotion",
     "business_id": "BIZ-001",
     "name": "Promosi",
     "display_order": 0,
     "is_active": true
   }
   ```
3. Jika seluruh promosi telah selesai (`ended`/`stopped`), grup virtual ini otomatis hilang dari respons.

### 5.2 Mengambil Produk dalam Grup Promosi
Frontend cukup mengirimkan query parameter `group_id=promotion`:
- **Customer:** `GET /customer/businesses/:business_id/products?group_id=promotion`
- **Merchant:** `GET /business/:business_id/products?product_group_id=promotion`

Backend akan mengeksekusi subquery internal untuk mengambil seluruh produk yang saat ini sedang tergabung dalam promosi aktif toko tersebut lengkap dengan harga diskon aslinya.

---

## 6. Integrasi Postingan Feed Otomatis

Setiap kali promosi dibuat, sistem secara otomatis menerbitkan satu baris postingan di tabel `business_posts` dengan atribut khusus:
- `is_promotion: true`
- `promotion_id: "PRM-xxxx"`
- `badge_text: "Promosi"`
- `target_action: "open_promotion_group"`
- `images`: Berisi foto preview dari `main_product` yang dipilih merchant.
- `content`: Berisi teks `caption` promosi.

Ketika customer mengklik postingan berlabel promosi di timeline/feed:
1. Frontend memeriksa `is_promotion == true`.
2. Frontend mengarahkan (*deep link*) customer ke halaman etalase toko dengan filter otomatis aktif pada tab grup `"Promosi"`.
3. Frontend mengirimkan event klik ke `POST /customer/promotions/events` (`source: "post"`).

---

## 7. Mekanisme Notifikasi & Deduplikasi

### 7.1 Sasaran Penerima
Ketika promosi aktif, notifikasi dipicu ke 2 kelompok customer:
1. Semua customer yang telah melakukan **Bookmark** ke toko tersebut (`business_bookmarks`).
2. Semua customer yang telah menambahkan salah satu produk promosi ke **Wishlist** (`product_wishlists`).

Sistem melakukan penyatuan set (*Set Union*) berbasis `account_id` di database sehingga customer yang mem-bookmark toko sekaligus mem-wishlist produknya **hanya akan menerima 1 notifikasi** (bebas duplikasi).

### 7.2 Format Notifikasi
- **Judul:** `🏷️ Promosi Baru dari [Nama Toko]`
- **Body:** Teks caption promosi yang ditulis oleh merchant (dipotong s.d 150 karakter).
- **Gambar (FCM & In-App):** URL foto preview produk utama (`main_product`).
- **Data Payload:**
  ```json
  {
    "type": "promotion",
    "promotion_id": "PRM-xxxx",
    "business_id": "BIZ-xxxx",
    "image_url": "https://s3.../product_preview.jpg",
    "action": "open_promotion_group"
  }
  ```

---

## 8. Background Scheduler & Worker

Worker berjalan di latar belakang setiap **1 menit** (`promotion.StartPromotionWorker`):
1. **Pemeriksaan Promosi Terjadwal (`scheduled` -> `active`):**
   - Mencari promosi dengan status `scheduled` yang `start_at <= NOW`.
   - Mengubah status menjadi `active`.
   - Menerapkan diskon ke produk di katalog.
   - Mengirim notifikasi ke pelanggan secara otomatis.
2. **Pemeriksaan Promosi Berakhir (`active` -> `ended`):**
   - Mencari promosi dengan status `active` yang `end_at < NOW`.
   - Mengubah status menjadi `ended`.
   - Memulihkan (*rollback*) seluruh diskon produk ke kondisi sebelum promosi dimulai.

---

## 9. Ringkasan Kode Respons API

| HTTP Status | Response Code | Arti |
|---|---|---|
| `200 OK` | `00` | Permintaan berhasil diproses. |
| `201 Created` | `00` | Promosi baru berhasil dibuat. |
| `400 Bad Request` | `01` / `02` | Validasi gagal (misal caption > 150 karakter, durasi > 90 hari, diskon melebihi harga). |
| `401 Unauthorized` | `03` | Token tidak disertakan atau sudah kedaluwarsa. |
| `403 Forbidden` | `03` | Akses ditolak (belum KYC, atau staf tidak memiliki izin `manage_promotions`). |
| `404 Not Found` | `04` | Promosi atau produk tidak ditemukan pada toko tersebut. |
| `500 Server Error` | `99` | Terjadi kendala internal server atau database. |
