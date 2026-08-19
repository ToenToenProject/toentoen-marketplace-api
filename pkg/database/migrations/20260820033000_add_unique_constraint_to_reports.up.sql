-- Hapus duplikat laporan jika ada sebelum membuat unique index
DELETE FROM reports a USING reports b
WHERE a.created_at < b.created_at
  AND a.account_id = b.account_id
  AND a.target_id = b.target_id;

-- Buat Unique Index untuk memastikan 1 akun hanya bisa melaporkan 1 target sekali
CREATE UNIQUE INDEX IF NOT EXISTS idx_reports_account_target ON reports(account_id, target_id);
