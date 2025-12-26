package network

// File: network/bridge.go
// Deskripsi:
// File ini adalah jembatan (bridge) antara kode Go dan kode C (CGo).
// Fungsinya:
// 1. Mengimpor file header C yang relevan.
// 2. Membungkus fungsi C native agar aman dipanggil dari Go.
// 3. Mengkonversi tipe data Go ke C (misal: string Go ke char* C) dan sebaliknya.
// 4. Menangani manajemen memori manual jika diperlukan oleh kode C.
