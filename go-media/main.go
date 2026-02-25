package main

import (
"database/sql"
"fmt"
"log"
"net/http"
"path/filepath"

"github.com/gin-gonic/gin"
_ "github.com/lib/pq"
)

const (
host     = "localhost"
port     = 5432
user     = "postgres"
password = "password123"
dbname   = "postgres"
)

func main() {
// 1. Setup Koneksi Database
psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
host, port, user, password, dbname)

db, err := sql.Open("postgres", psqlInfo)
if err != nil {
	log.Fatal(err)
}
defer db.Close()

err = db.Ping()
if err != nil {
	log.Fatal("Tidak bisa konek ke database:", err)
}
fmt.Println("Berhasil konek ke PostgreSQL dari Go!")

// 2. Inisialisasi Gin Router
r := gin.Default()

	// --- TAMBAHKAN BLOK CORS INI ---
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})
// 3. FITUR STREAMING: Menyajikan file statis
// Ini membuat aplikasi Frontend nanti bisa memutar lagu langsung pakai URL
r.Static("/media", "./uploads")

// 4. API Endpoint: Cek status server
r.GET("/ping", func(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Media Service Go Menyala dan Siap Streaming!"})
})

// 5. API Endpoint: Upload Lagu
r.POST("/songs", func(c *gin.Context) {
	title := c.PostForm("title")
	artist := c.PostForm("artist")

	// Ambil file mp3 dari request
	file, err := c.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File audio tidak ditemukan"})
		return
	}

	// Tentukan nama dan lokasi simpan
	filename := filepath.Base(file.Filename)
	savePath := fmt.Sprintf("uploads/songs/%s", filename)
	
	// URL ini yang akan disimpan di database & dipakai Frontend untuk nge-play lagu
	fileURL := fmt.Sprintf("/media/songs/%s", filename) 

	// Simpan file fisik ke dalam folder laptop
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan file ke disk"})
		return
	}

	// Simpan datanya (Judul, Artis, URL) ke PostgreSQL
	sqlStatement := `INSERT INTO songs (title, artist, file_url) VALUES ($1, $2, $3) RETURNING id`
	var id string
	err = db.QueryRow(sqlStatement, title, artist, fileURL).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan ke database"})
		return
	}

	// Beri respon sukses
	c.JSON(http.StatusOK, gin.H{
		"message":  "Lagu berhasil diupload dan disimpan!",
		"song_id":  id,
		"file_url": fileURL,
	})
})

// Jalankan di Port 8080 agar tidak bentrok dengan Python (8000)
fmt.Println("Server Go berjalan di http://localhost:8080")
r.Run(":8080")
}