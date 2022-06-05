package legacy

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func getContentTypeOfFile(ext string) string {
	images := []string{".bmp", ".dib", ".gif", ".heif", ".heic", ".jpg", ".jpeg", ".jpe", ".jif", ".jfif", ".jfi", ".jp2", ".j2k", ".jpf", ".jpx", ".jpm", ".mj2", ".png", ".svg", ".svgz", ".tiff", ".tif", ".webp"}
	for _, b := range images {
		if b == ext {
			return "image"
		}
	}
	return "document"
}

func goDotEnvVariable(key string) string {

	err := godotenv.Load(".env")

	if err != nil {
		log.Println("Error loading .env file")
	}

	return os.Getenv(key)
}
