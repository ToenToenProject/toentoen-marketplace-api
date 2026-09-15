# Dokumentasi Lengkap: Product CRUD API & Sistem Diskon

Dokumentasi ini menjelaskan secara komprehensif seluruh endpoint CRUD (Create, Read, Update, Delete) untuk produk merchant, termasuk detail spesifikasi sistem diskon fleksibel (Nominal & Persentase dengan Maksimal Diskon).

---

## 1. Konsep & Arsitektur Utama

### 1.1 Base URL & Autentikasi
* **Base URL**: `/api/v1/business/:business_id/products`
* **Autentikasi**: Wajib login. Sistem membaca token dari:
  1. Header HTTP: `Authorization: Bearer <token>`, ATAU
  2. Cookie: `auth_token=<token>`
* **Role Akses**:
  - `owner` (Pemilik bisnis)
  - `business_admin` (Admin toko yang memiliki permission terkait katalog)

### 1.2 Sistem Diskon Fleksibel
Merchant memiliki pilihan menentukan jenis diskon:
1. **`discount_type: "percentage"`**: Potongan berbasis persentase (misal: 20%).
   - Nilai `discount` berada di antara `0` hingga `100`.
   - **`max_discount`** *(opsional)*: Batas maksimal potongan dalam nominal Rupiah. Contoh: Diskon 50% Maksimal Rp 10.000.
2. **`discount_type: "nominal"`**: Potongan langsung dalam nominal Rupiah (misal: Rp 5.000).
   - Nilai `discount` tidak boleh melebihi harga produk (`discount <= price`).
3. **Tanpa Diskon (Harga Normal)**:
   - Nilai `discount: 0` (atau tidak dikirim).
   - Backend otomatis mereset `discount_type = "nominal"`, `discount = 0`, `discount_amount = 0`, `discount_pct = 0`, dan `is_discounted = false`.

---

## 2. Rangkuman Seluruh Endpoint Produk

| Metode | Endpoint | Deskripsi |
|---|---|---|
| `POST` | `/api/v1/business/:business_id/products` | Membuat produk baru (lengkap dengan diskon & stok) |
| `GET` | `/api/v1/business/:business_id/products` | Mengambil daftar produk (filter, search, sort, cursor pagination) |
| `GET` | `/api/v1/business/:business_id/products/:product_id` | Mengambil detail lengkap 1 produk |
| `PUT` | `/api/v1/business/:business_id/products/:product_id` | Mengupdate produk (bisa full atau partial update) |
| `DELETE` | `/api/v1/business/:business_id/products/:product_id` | Menghapus produk (soft delete) |
| `PATCH` | `/api/v1/business/:business_id/products/bulk-status` | Mengaktifkan/menonaktifkan banyak produk sekaligus |
| `DELETE` | `/api/v1/business/:business_id/products/bulk-delete` | Menghapus banyak produk sekaligus |
| `POST` | `/api/v1/business/:business_id/products/:product_id/variants` | Menambahkan varian produk baru |
| `PATCH` | `/api/v1/business/:business_id/products/:product_id/variants/:variant_id` | Mengupdate varian produk |
| `DELETE` | `/api/v1/business/:business_id/products/:product_id/variants/:variant_id` | Menghapus varian produk |

---

## 3. Detail Endpoint CRUD

### 3.1 Create Product (Buat Produk)
* **Endpoint**: `POST /api/v1/business/:business_id/products`
* **Content-Type**: `application/json`

#### Spesifikasi Field Request:
| Field | Tipe | Wajib? | Keterangan |
|---|---|---|---|
| `product_name` | `string` | **Ya** | Nama produk (maksimal 255 karakter) |
| `price` | `number` | **Ya** | Harga dasar produk (minimal 0) |
| `discount_type` | `string` | Tidak | `"percentage"` atau `"nominal"` (default: `"nominal"`) |
| `discount` | `number` | Tidak | Nilai diskon (Rupiah jika nominal, angka 1-100 jika percentage) |
| `max_discount` | `number` | Tidak | Batas maksimal potongan Rupiah (khususnya untuk persentase) |
| `discount_start_at`| `string (ISO8601)` | Tidak | Waktu mulai diskon aktif (contoh: `"2026-09-15T00:00:00Z"`) |
| `discount_end_at` | `string (ISO8601)` | Tidak | Waktu akhir diskon aktif (contoh: `"2026-09-30T23:59:59Z"`) |
| `product_group_id`| `string` | Tidak | ID grup/kategori produk toko |
| `product_preview` | `array of string` | Tidak | Daftar URL gambar produk (maksimal 10 gambar) |
| `product_description`| `string`| Tidak | Deskripsi produk (maksimal 300 karakter) |
| `product_origin` | `string` | Tidak | `"domestic"` atau `"imported"` (default: `"domestic"`) |
| `stock_type` | `string` | Tidak | `"in_stock"`, `"out_of_stock"`, `"limited"` |
| `stock_quantity` | `integer` | Tidak | Jumlah stok fisik produk (default: 0) |
| `low_stock_threshold`| `integer`| Tidak | Batas peringatan stok menipis (default: 5) |
| `tags` | `array of string` | Tidak | Tag produk untuk pencarian |
| `is_active` | `boolean` | Tidak | Status aktif/tidaknya produk di toko (default: true) |

---

### 3.2 Tiga Skenario Diskon (Request & Response)

#### Skenario 1: Diskon Persentase (%) dengan Maksimal Potongan (`max_discount`)
Merchant memasang diskon **50%** dengan batas maksimal potongan **Rp 10.000** pada produk seharga **Rp 50.000**.

**Request Body (`POST /products`):**
```json
{
    "product_group_id": "GRP-20260710141403-090DGQ",
    "product_name": "Es Kelapa Spesial",
    "product_description": "Es kelapa muda segar dengan sirup gula aren",
    "price": 50000,
    "discount_type": "percentage",
    "discount": 50,
    "max_discount": 10000,
    "product_preview": [
        "https://storage.toentoen.id/eskelapa-1.jpg"
    ],
    "stock_type": "in_stock",
    "stock_quantity": 100,
    "low_stock_threshold": 10,
    "tags": ["Es Kelapa", "Promo"],
    "is_active": true
}
```

**Response Sukses (`201 Created`):**
```json
{
    "status": "success",
    "message": "Produk berhasil dibuat",
    "data": null
}
```

**Saat Detail Produk Diambil (`GET /products/:product_id`):**
```json
{
    "status": "success",
    "message": "Detail produk berhasil diambil",
    "data": {
        "product_id": "PRD-20260915150000-A1B2C3",
        "business_id": "BUS-20260911042959-O2C3UH",
        "product_name": "Es Kelapa Spesial",
        "price": 50000,
        "original_min_price": 50000,
        "original_max_price": 50000,
        "final_min_price": 40000,
        "final_max_price": 40000,
        "is_discounted": true,
        "discount_type": "percentage",
        "discount": 50,
        "discount_amount": 10000,
        "discount_pct": 20.0,
        "max_discount": 10000,
        "stock_type": "in_stock",
        "stock_quantity": 100,
        "low_stock_threshold": 10,
        "is_active": true
    }
}
```
* **Keterangan**: 50% dari Rp 50.000 adalah Rp 25.000. Karena ada `max_discount: 10000`, maka potongan harga dibatasi tepat menjadi **Rp 10.000**, harga bayar menjadi **Rp 40.000**, dan `discount_pct` efektif bernilai **20%**.

---

#### Skenario 2: Diskon Nominal (Rupiah Langsung)
Merchant memasang potongan harga langsung sebesar **Rp 5.000** pada produk seharga **Rp 25.000**.

**Request Body (`POST /products`):**
```json
{
    "product_name": "Es Kelapa Original",
    "price": 25000,
    "discount_type": "nominal",
    "discount": 5000,
    "stock_type": "in_stock",
    "stock_quantity": 50
}
```

**Saat Detail Produk Diambil (`GET /products/:product_id`):**
```json
{
    "status": "success",
    "message": "Detail produk berhasil diambil",
    "data": {
        "product_id": "PRD-20260915150000-D4E5F6",
        "product_name": "Es Kelapa Original",
        "price": 25000,
        "original_min_price": 25000,
        "final_min_price": 20000,
        "is_discounted": true,
        "discount_type": "nominal",
        "discount": 5000,
        "discount_amount": 5000,
        "discount_pct": 20.0,
        "stock_type": "in_stock",
        "stock_quantity": 50,
        "is_active": true
    }
}
```
* **Keterangan**: Potongan harga riil (`discount_amount`) adalah **Rp 5.000**, dan backend secara otomatis menghitung `discount_pct` sebesar **20%** (`Rp 5.000 / Rp 25.000 * 100%`).

---

#### Skenario 3: Tanpa Diskon (Harga Normal)
Merchant menjual produk dengan harga normal tanpa diskon.

**Request Body (`POST /products`):**
```json
{
    "product_name": "Es Kelapa Biasa",
    "price": 25000,
    "discount": 0,
    "stock_type": "in_stock"
}
```
*(Catatan: `discount_type` dan `max_discount` tidak perlu dikirim)*

**Saat Detail Produk Diambil (`GET /products/:product_id`):**
```json
{
    "status": "success",
    "message": "Detail produk berhasil diambil",
    "data": {
        "product_id": "PRD-20260915150000-G7H8I9",
        "product_name": "Es Kelapa Biasa",
        "price": 25000,
        "original_min_price": 25000,
        "final_min_price": 25000,
        "is_discounted": false,
        "discount_type": "nominal",
        "discount": 0,
        "discount_amount": 0,
        "discount_pct": 0,
        "stock_type": "in_stock",
        "is_active": true
    }
}
```
* **Keterangan**: `is_discounted: false`, harga coret sama dengan harga bayar (Rp 25.000).

---

### 3.3 Update Product (Edit Produk)
* **Endpoint**: `PUT /api/v1/business/:business_id/products/:product_id`
* **Content-Type**: `application/json`

Endpoint ini mendukung **Full Update** maupun **Partial Update** (hanya mengirim field yang hendak diganti).

#### Contoh 1: Mengubah Menjadi Diskon Persentase + Max Cap
```json
{
    "discount_type": "percentage",
    "discount": 30,
    "max_discount": 15000
}
```

#### Contoh 2: Mengubah Menjadi Diskon Nominal
```json
{
    "discount_type": "nominal",
    "discount": 7000
}
```

#### Contoh 3: Menghapus Diskon (Kembali ke Harga Normal)
Cukup kirimkan nilai `discount: 0`:
```json
{
    "discount": 0
}
```
> [!TIP]
> Backend secara otomatis mereset `max_discount = null`, `discount_start_at = null`, dan `discount_end_at = null` saat menerima `discount: 0`.

---

### 3.4 Get All Products (Daftar Produk Merchant)
* **Endpoint**: `GET /api/v1/business/:business_id/products`
* **Query Parameters**:
  - `search`: Kata kunci pencarian nama atau tag produk
  - `product_group_id`: Filter ID grup produk
  - `stock_type`: Filter stok (`"in_stock"`, `"out_of_stock"`, `"limited"`)
  - `is_active`: Filter status aktif (`true` atau `false`)
  - `sort`: Urutan (`"price_asc"`, `"price_desc"`, `"name_asc"`, `"newest"`)
  - `cursor`: Pointer product_id untuk infinite scroll pagination
  - `limit`: Jumlah data per batch (default: 20)

#### Contoh Response:
```json
{
    "status": "success",
    "message": "Daftar produk berhasil diambil",
    "data": {
        "products": [
            {
                "product_id": "PRD-20260915150000-A1B2C3",
                "product_name": "Es Kelapa Spesial",
                "image_url": "https://storage.toentoen.id/eskelapa-1.jpg",
                "has_variants": false,
                "original_min_price": 50000,
                "original_max_price": 50000,
                "final_min_price": 40000,
                "final_max_price": 40000,
                "is_discounted": true,
                "discount_type": "percentage",
                "discount": 50,
                "discount_amount": 10000,
                "discount_pct": 20.0,
                "max_discount": 10000,
                "stock_type": "in_stock",
                "rating": 5.0,
                "like_count": 12,
                "review_count": 4
            }
        ],
        "summary": {
            "total_all": 25,
            "total_in_stock": 20,
            "total_pre_order": 0,
            "total_limited": 3,
            "total_out_of_stock": 2
        },
        "pagination": {
            "next_cursor": "PRD-20260915150000-A1B2C3",
            "has_more": true
        }
    }
}
```

---

### 3.5 Delete Product (Hapus Produk)
* **Endpoint**: `DELETE /api/v1/business/:business_id/products/:product_id`
* **Response Sukses (`200 OK`)**:
```json
{
    "status": "success",
    "message": "Produk berhasil dihapus",
    "data": null
}
```

---

### 3.6 Bulk Operations (Operasi Sekaligus)

#### Bulk Update Status Aktif:
* **Endpoint**: `PATCH /api/v1/business/:business_id/products/bulk-status`
* **Request Body**:
```json
{
    "product_ids": [
        "PRD-20260915150000-A1B2C3",
        "PRD-20260915150000-D4E5F6"
    ],
    "is_active": false
}
```

#### Bulk Delete:
* **Endpoint**: `DELETE /api/v1/business/:business_id/products/bulk-delete`
* **Request Body**:
```json
{
    "product_ids": [
        "PRD-20260915150000-A1B2C3",
        "PRD-20260915150000-D4E5F6"
    ]
}
```

---

### 3.7 Manajemen Varian Produk (Product Variants)
Jika produk memiliki opsi ukuran/rasa/pilihan:

1. **Tambah Varian**: `POST /api/v1/business/:business_id/products/:product_id/variants`
   ```json
   {
       "sku": "ESK-JUMBO",
       "variant_name": "Ukuran Jumbo",
       "additional_price": 5000,
       "stock_quantity": 30,
       "is_active": true
   }
   ```
2. **Update Varian**: `PATCH /api/v1/business/:business_id/products/:product_id/variants/:variant_id`
   ```json
   {
       "additional_price": 6000,
       "stock_quantity": 25
   }
   ```
3. **Hapus Varian**: `DELETE /api/v1/business/:business_id/products/:product_id/variants/:variant_id`

---

## 4. Validasi Error Standar

| Kondisi Error | Status Code | Pesan Error |
|---|---|---|
| `discount_type` bukan nominal / percentage | `400 Bad Request` | `"tipe diskon tidak valid, pilih 'nominal' atau 'percentage'"` |
| Diskon persen > 100% | `400 Bad Request` | `"persentase diskon tidak boleh melebihi 100%"` |
| Diskon nominal > harga produk | `400 Bad Request` | `"nominal diskon tidak boleh melebihi harga produk"` |
| Diskon bernilai negatif | `400 Bad Request` | `"nilai diskon tidak boleh bernilai negatif"` |
| `max_discount` bernilai negatif | `400 Bad Request` | `"maksimal diskon tidak boleh bernilai negatif"` |
| Harga dasar produk negatif | `400 Bad Request` | `"harga produk tidak boleh bernilai negatif"` |
| Nama produk kosong | `400 Bad Request` | `"Nama produk wajib diisi dan harga tidak boleh negatif"` |
