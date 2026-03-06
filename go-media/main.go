package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
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

	// Create songs table if it doesn't exist
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS songs (
		id SERIAL PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		artist VARCHAR(255) NOT NULL,
		file_url TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal("Gagal membuat tabel songs:", err)
	}
	fmt.Println("Tabel songs siap!")

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

	// Create uploads/songs directory if not exists
	os.MkdirAll("uploads/songs", 0755)
	fmt.Println("Directory uploads/songs ready!")

	// 4. API Endpoint: Cek status server
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Media Service Go Menyala dan Siap Streaming!"})
	})

	// 5. API Endpoint: Get All Songs
	r.GET("/songs", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, title, artist, file_url FROM songs ORDER BY id DESC")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data lagu"})
			return
		}
		defer rows.Close()

		songs := []gin.H{}
		for rows.Next() {
			var id int
			var title, artist, fileURL string
			if err := rows.Scan(&id, &title, &artist, &fileURL); err != nil {
				continue
			}
			songs = append(songs, gin.H{
				"id":       id,
				"title":    title,
				"artist":   artist,
				"file_url": fileURL,
			})
		}

		c.JSON(http.StatusOK, gin.H{"songs": songs})
	})

	// 6. API Endpoint: Upload Lagu
	r.POST("/songs", func(c *gin.Context) {
		title := c.PostForm("title")
		artist := c.PostForm("artist")

		fmt.Printf("[UPLOAD] Title: %s, Artist: %s\n", title, artist)

		// Ambil file mp3 dari request
		file, err := c.FormFile("audio")
		if err != nil {
			fmt.Printf("[ERROR] File retrieval failed: %v\n", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "File audio tidak ditemukan"})
			return
		}
		fmt.Printf("[UPLOAD] File received: %s (size: %d bytes)\n", file.Filename, file.Size)

		// Tentukan nama dan lokasi simpan
		filename := filepath.Base(file.Filename)
		savePath := fmt.Sprintf("uploads/songs/%s", filename)

		// URL ini yang akan disimpan di database & dipakai Frontend untuk nge-play lagu
		fileURL := fmt.Sprintf("/media/songs/%s", filename)

		// Simpan file fisik ke dalam folder laptop
		if err := c.SaveUploadedFile(file, savePath); err != nil {
			fmt.Printf("[ERROR] File save failed: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan file ke disk: " + err.Error()})
			return
		}
		fmt.Printf("[UPLOAD] File saved to: %s\n", savePath)

		// Simpan datanya (Judul, Artis, URL) ke PostgreSQL
		sqlStatement := `INSERT INTO songs (title, artist, file_url) VALUES ($1, $2, $3) RETURNING id`
		var id string // Change to string to handle UUID
		err = db.QueryRow(sqlStatement, title, artist, fileURL).Scan(&id)
		if err != nil {
			fmt.Printf("[ERROR] Database insert failed: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan ke database: " + err.Error()})
			return
		}
		fmt.Printf("[SUCCESS] Song saved with ID: %s\n", id)

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
