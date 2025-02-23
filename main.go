package main

import (
    "database/sql"
    "io"
    "log"
    "net/http"
    "os"

    "github.com/gin-gonic/gin"
    _ "github.com/go-sql-driver/mysql"
    "github.com/google/uuid"
)

type Album struct {
    Artist string `json:"artist"`
    Title  string `json:"title"`
    Year   string `json:"year"`
}

type PostResponse struct {
    AlbumID   string `json:"albumID"`
    ImageSize string `json:"imageSize"`
}

var db *sql.DB

func main() {
    dsn := os.Getenv("DB_DSN")
    if dsn == "" {
        log.Fatal("DB_DSN environment variable not set")
    }

    var err error
    db, err = sql.Open("mysql", dsn)
    if err != nil {
        log.Fatalf("Failed to open DB: %v", err)
    }

    err = db.Ping()
    if err != nil {
        log.Fatalf("Failed to connect to DB: %v", err)
    }

    // 确保数据库表结构正确，仅使用 albumID 作为唯一标识
    db.Exec(`
        CREATE TABLE IF NOT EXISTS albums (
            albumID VARCHAR(36) NOT NULL UNIQUE,
            image LONGBLOB,
            profile JSON
        ) ENGINE=InnoDB;
    `)

    r := gin.Default()

    // 测试路由/health
    r.GET("/health", healthCheck)

    // 新增 API 路由
    r.POST("/albums", doPost)
    r.GET("/albums/:albumID", goGet)

    port := os.Getenv("PORT")
    if port == "" {
        port = "8081"
    }

    log.Printf("Server starting on port %s ...", port)
    r.Run(":" + port)
}

// healthCheck: 返回 200 OK
func healthCheck(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// doPost: 保存图片的 Binary 数据和 profile 数据到数据库
func doPost(c *gin.Context) {
    file, _, err := c.Request.FormFile("image")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Image file is required"})
        return
    }
    defer file.Close()

    // 读取图片文件的 Binary 数据
    fileBytes, err := io.ReadAll(file)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read image file"})
        return
    }

    // 获取 profile JSON 数据
    profileData := c.PostForm("profile")

    // 生成 UUID 作为 albumID
    albumID := uuid.New().String()

    // 将图片 Binary 数据和 profile 数据插入数据库
    query := "INSERT INTO albums (albumID, image, profile) VALUES (?, ?, ?)"
    _, err = db.Exec(query, albumID, fileBytes, profileData)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert album into database"})
        return
    }

    c.JSON(http.StatusCreated, PostResponse{
        AlbumID:   albumID,
        ImageSize: string(len(fileBytes)),
    })
}

// goGet: 根据 albumID 返回 profile 数据 (不返回图片 Binary)
func goGet(c *gin.Context) {
    albumID := c.Param("albumID")
    if albumID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Album ID is required"})
        return
    }

    var profileData string
    query := "SELECT profile FROM albums WHERE albumID = ?"
    err := db.QueryRow(query, albumID).Scan(&profileData)
    if err != nil {
        if err == sql.ErrNoRows {
            c.JSON(http.StatusNotFound, gin.H{"error": "Album not found"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch album"})
        }
        return
    }

    c.Data(http.StatusOK, "application/json", []byte(profileData))
}

