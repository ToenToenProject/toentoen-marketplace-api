# Proposal & Panduan Kebijakan: Pengaturan Hak Akses Admin Bisnis (Staff)

---

## 1. Ringkasan Eksekutif
Dalam rangka mendukung pertumbuhan operasional mitra merchant di ekosistem **ToenToen**, fitur **Delegasi Wewenang & Kontrol Akses (Role-Based Access Control)** dirancang agar Pemilik Bisnis (*Owner*) dapat mendelegasikan tugas harian kepada staf/karyawan (*Business Admin*) dengan aman, terukur, dan transparan.

Sistem ini memastikan:
1. **Efisiensi Operasional**: Staf dapat menjalankan tugasnya tanpa meminta akses akun utama pemilik.
2. **Mitigasi Risiko**: Mencegah kesalahan operasional atau penyalahgunaan wewenang pada data sensitif dan finansial toko.
3. **Transparansi Penuh**: Setiap perubahan wewenang akan tercatat dan dinotifikasikan secara terperinci kepada staf secara *real-time*.

---

## 2. Struktur Hirarki Akun Bisnis

| Peran (*Role*) | Deskripsi | Batasan Akses |
| :--- | :--- | :--- |
| **Pemilik Toko (*Owner*)** | Pemegang hak tertinggi, entitas hukum, dan pemilik modal bisnis. | **Akses Penuh (100%)** mencakup finansial, legalitas, dan kepemilikan toko. |
| **Supervisor** | Manajer operasional kepercayaan pemilik toko. | Akses operasional penuh secara *default* untuk mengawasi seluruh aktivitas staf. |
| **Staf Operasional (*Business Admin*)** | Karyawan/staf dengan tugas operasional spesifik (misal: kasir, staf gudang, CS). | **Akses Terbatas & Fleksibel** sesuai izin per fitur yang diberikan oleh Pemilik Toko. |

---

## 3. Matriks Fitur Operasional & Tingkat Wewenang Staf

Pemilik toko dapat mengaktifkan atau menonaktifkan masing-masing fitur berikut untuk setiap staf:

### A. Pengelolaan Produk & Etalase Toko
| Fitur Bisnis | Deskripsi Manfaat | Tingkat Risiko | Rekomendasi Default |
| :--- | :--- | :---: | :---: |
| **Tambah Produk Baru** | Menambah item atau menu baru ke katalog toko. | Sedang | ✅ Diizinkan |
| **Ubah Informasi Produk** | Mengubah nama, deskripsi, harga jual, diskon, dan tag produk. | Sedang | ✅ Diizinkan |
| **Kelola Foto & Media Produk** | Mengunggah dan memperbarui foto/galeri produk agar menarik. | Rendah | ✅ Diizinkan |
| **Hapus Produk** | Menghapus item dari katalog (permanen atau arsip). | **Tinggi** | ❌ Dinonaktifkan |
| **Atur Kategori & Grup Produk** | Mengelompokkan produk ke kategori tertentu (misal: "Menu Favorit", "Paket Hemat"). | Rendah | ✅ Diizinkan |

---

### B. Jam Kerja & Operasional Toko
| Fitur Bisnis | Deskripsi Manfaat | Tingkat Risiko | Rekomendasi Default |
| :--- | :--- | :---: | :---: |
| **Ubah Profil Toko** | Memperbarui deskripsi toko, kontak narahubung, tautan medsos, dan banner toko. | Sedang | ❌ Dinonaktifkan |
| **Atur Jadwal Buka/Tutup** | Mengatur jam operasional rutin mingguan toko. | Sedang | ❌ Dinonaktifkan |
| **Buka/Tutup Toko Sementara** | Menutup toko secara cepat saat darurat atau jeda istirahat (*break*). | Sedang | ✅ Diizinkan |
| **Aktivasi Live Shopping** | Mengaktifkan status toko sedang siaran langsung (*live selling*). | Rendah | ✅ Diizinkan |

---

### C. Komunikasi & Layanan Pelanggan (*Customer Care*)
| Fitur Bisnis | Deskripsi Manfaat | Tingkat Risiko | Rekomendasi Default |
| :--- | :--- | :---: | :---: |
| **Chat Langsung Pelanggan** | Membaca dan merespons pertanyaan pembeli secara *real-time*. | Rendah | ✅ Diizinkan |
| **Balas Ulasan Toko** | Menjawab ulasan, pujian, atau masukan dari pembeli di halaman toko. | Rendah | ✅ Diizinkan |
| **Respon Permintaan Pasar (*Demand Signals*)** | Menawarkan produk toko ke calon pembeli di sekitar yang sedang mencari kebutuhan tertentu. | Sedang | ✅ Diizinkan |

---

### D. Pemasaran & Konten Feed Toko
| Fitur Bisnis | Deskripsi Manfaat | Tingkat Risiko | Rekomendasi Default |
| :--- | :--- | :---: | :---: |
| **Kelola Postingan Feed Toko** | Mengunggah cerita, promosi singkat, atau konten edukasi di feed toko. | Rendah | ✅ Diizinkan |

---

### E. Laporan & Analitik Toko
| Fitur Bisnis | Deskripsi Manfaat | Tingkat Risiko | Rekomendasi Default |
| :--- | :--- | :---: | :---: |
| **Lihat Dashboard Analitik** | Memantau statistik kunjungan harian toko, tren produk terlaris, dan histori transaksi. | Sedang | ❌ Dinonaktifkan |

---

### F. Pusat Bantuan & Eskalasi Platform
| Fitur Bisnis | Deskripsi Manfaat | Tingkat Risiko | Rekomendasi Default |
| :--- | :--- | :---: | :---: |
| **Buat Tiket Bantuan ke ToenToen** | Menghubungi tim dukungan platform resmi jika toko mengalami kendala teknis. | Rendah | ✅ Diizinkan |

---

## 4. Proteksi Khusus: Hak Mutlak Pemilik (*Owner-Only Privileges*)

Untuk menjaga keamanan aset bisnis, fitur-fitur di bawah ini **TIDAK PERNAH DAPAT DIDELEGASIKAN** kepada staf mana pun:

1. **Penutupan / Penghapusan Toko**: Pembubaran toko hanya sah dilakukan oleh pemilik resmi.
2. **Pengalihan Kepemilikan Toko (*Store Ownership Transfer*)**: Proses pemindahan hak milik bisnis membutuhkan validasi OTP dan verifikasi identitas (KYC) pemilik.
3. **Manajemen Staf & Pengaturan Gaji/Role**: Hanya pemilik yang berhak mengangkat, mencopot, atau mengubah hak akses staf lain.
4. **Rekening Finansial & Penarikan Saldo**: Rekening pencairan dana (*payout*) terikat langsung pada rekening bank terverifikasi milik *Owner*.

---

## 5. Sistem Notifikasi Transparan ke Staf (*Audit & Governance*)

Setiap kali pemilik toko menambahkan atau mencabut wewenang seorang staf, sistem secara otomatis mengirimkan notifikasi resmi ke akun staf tersebut.

### Format Notifikasi yang Diterima Staf:
> **Pembaruan Hak Akses Toko**  
> *Pemilik bisnis telah memperbarui wewenang kerja Anda di **Toko Kopi Sejahtera**:*  
>  
> 🟢 **Wewenang Baru Ditambahkan:**  
> • *Chat Langsung Pelanggan*  
> • *Balas Ulasan Toko*  
>  
> 🔴 **Wewenang Dicabut:**  
> • *Hapus Produk*  
>  
> *Pembaruan ini berlaku efektif saat ini. Silakan hubungi pemilik bisnis jika ada pertanyaan.*

---

## 6. Contoh Templat Penerapan di Lapangan

Untuk mempermudah pemilik toko, sistem dapat menyediakan opsi profil siap pakai (*Preset Template*):

1. **Profil: Customer Service / Kasir**
   - *Fokus*: Pelayanan pelanggan dan kelancaran transaksi.
   - *Akses*: Chat Pelanggan, Balas Ulasan Toko, Respon Demand Signals, Buka/Tutup Toko Sementara.
2. **Profil: Admin Gudang & Katalog**
   - *Fokus*: Manajemen etalase dan ketersediaan barang.
   - *Akses*: Tambah Produk Baru, Ubah Informasi Produk, Kelola Foto Produk, Atur Kategori Produk.
3. **Profil: Marketing & Konten**
   - *Fokus*: Promosi dan branding toko.
   - *Akses*: Kelola Postingan Feed Toko, Mode Live Shopping, Kelola Foto Produk.
