-- Beri kuota meme nyata per paket: 0 (unlimited) hanya untuk paket termahal.
-- Tanpa ini, semua paket unlimited sehingga bug race kuota (TKT-109) tidak
-- pernah bisa direproduksi di paket mana pun.
UPDATE plans SET max_memes = 10  WHERE code = 'premium_1d';
UPDATE plans SET max_memes = 25  WHERE code = 'premium_3d';
UPDATE plans SET max_memes = 100 WHERE code = 'premium_30d';
UPDATE plans SET max_memes = 250 WHERE code = 'premium_60d';
-- premium_90d tetap unlimited (max_memes = 0)