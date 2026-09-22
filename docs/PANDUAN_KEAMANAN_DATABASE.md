# 🛡️ Panduan Keamanan Database ToenToen

Dokumen ini menjelaskan konfigurasi dan langkah-langkah implementasi untuk memastikan:
1. **Database Tidak Terbuka ke Publik** *(Private & Isolated)*.
2. **Hak Akses Database Ketat** *(Principle of Least Privilege)*.

---

## 🔒 1. Mengamankan Database dari Akses Publik

### Masalah Sebelumnya
Sebelumnya di `docker-compose.yaml`, port database dipetakan dengan `"5433:5432"`. Format ini secara otomatis mem-binding port ke `0.0.0.0`, yang berarti port terbuka ke seluruh antarmuka jaringan publik internet.

### Solusi yang Diterapkan di `docker-compose.yaml`
Port internal (PostgreSQL, Redis, Meilisearch) kini dikunci hanya ke `127.0.0.1` (Localhost Server):

```yaml
services:
  postgres-db:
    # ...
    ports:
      - "127.0.0.1:5433:5432" # ✅ HANYA bisa diakses dari lokal host server

  redis-cache:
    # ...
    ports:
      - "127.0.0.1:6379:6379" # ✅ HANYA bisa diakses dari lokal host server

  meilisearch:
    # ...
    ports:
      - "127.0.0.1:7700:7700" # ✅ HANYA bisa diakses dari lokal host server
```

### Dampak Positif
* **Koneksi Antar-Container Tetap Normal**: `api-server` berkomunikasi via jaringan internal Docker (`postgres-db:5432`), sehingga performa dan fungsionalitas API tidak terganggu.
* **Tertutup dari Internet Publik**: Port 5433, 6379, dan 7700 tidak lagi merespons request dari luar server EC2.
* **Akses Developer Tetap Aman via SSH Tunnel**: Jika tim pengembang ingin membuka database via DBeaver / TablePlus / pgAdmin dari laptop, cukup gunakan SSH Tunneling:
  ```bash
  ssh -i "toentoen-key-ec2.pem" -L 5433:127.0.0.1:5433 ubuntu@IP_SERVER_EC2
  ```
  Lalu di DBeaver connect ke `localhost:5433`.

---

## 🔑 2. Pengetatan Hak Akses Database (*Least Privilege*)

### Prinsip Keamanan
Aplikasi backend runtime (`api-server`) **tidak boleh** berjalan dengan user superuser / root database. Akun aplikasi hanya boleh melakukan manipulasi data (DML: `SELECT`, `INSERT`, `UPDATE`, `DELETE`), dan **dilarang keras** memiliki izin mengubah struktur tabel (DDL: `DROP TABLE`, `ALTER TABLE`, `TRUNCATE`, `DROP DATABASE`).

### Arsitektur Dua Akun Database:
1. **Akun Migrator / Superuser (Admin DB)**:
   * Digunakan oleh service `migrator` saat proses deployment untuk membuat/mengubah tabel (`migrate up`).
2. **Akun Aplikasi (`toentoen_app`)**:
   * Digunakan oleh `api-server` untuk melayani request transaksi user sehari-hari.
   * Hanya memiliki izin `SELECT, INSERT, UPDATE, DELETE` di skema `public`.

---

## 🚀 Langkah Eksekusi di Server EC2

### Langkah 1: Tarik Kode Terbaru
```bash
cd ~/webapi-go
git pull origin main
```

### Langkah 2: Eksekusi Skrip Penguncian Hak Akses
Jalankan skrip [scripts/harden_db_permissions.sql](file:///d:/ToenToen/webapi-go/scripts/harden_db_permissions.sql) ke dalam container PostgreSQL:

```bash
# 1. Masuk ke direktori webapi-go
cd ~/webapi-go

# 2. Jalankan skrip SQL menggunakan user superuser yang ada di .env
source .env
sudo docker exec -i toentoen_postgres psql -U $DB_USER -d $DB_NAME < scripts/harden_db_permissions.sql
```

*(Catatan: Anda dapat mengubah password akun `toentoen_app` di dalam skrip `scripts/harden_db_permissions.sql` sebelum mengeksekusinya).*

### Langkah 3: Update `.env` untuk `api-server`
Ubah kredensial database di `.env` server agar menggunakan akun `toentoen_app`:
```env
DB_USER=toentoen_app
DB_PASSWORD=ToenToenAppSecurePass2026!
ENABLE_AUTO_MIGRATE=false
DB_SSLMODE=disable
```

> **💡 Catatan Penting:** 
> Untuk service `migrator`, pastikan tetap menggunakan kredensial superuser saat menjalankan migrasi:
> ```bash
> sudo docker-compose run --rm migrator
> ```

### Langkah 4: Restart Container
```bash
sudo docker-compose up -d --build api-server postgres-db redis-cache meilisearch
```

---

## 🧪 Verifikasi Keamanan

1. **Uji Port Publik dari Komputer Luar:**
   Jalankan di Command Prompt / Terminal laptop Anda:
   ```bash
   telnet IP_SERVER_EC2 5433
   ```
   **Hasil yang Diharapkan:** *Connection refused* atau *Timed out* (port tertutup rapat dari luar).

2. **Uji Hak Akses Terbatas (`toentoen_app`):**
   Login ke database sebagai `toentoen_app`:
   ```bash
   sudo docker exec -it toentoen_postgres psql -U toentoen_app -d toentoen_db
   ```
   Coba jalankan perintah berbahaya:
   ```sql
   DROP TABLE accounts;
   ```
   **Hasil yang Diharapkan:**
   ```
   ERROR: must be owner of table accounts
   ```
   Tabel aman dari penghapusan tidak disengaja maupun eksploitasi celah injeksi!
