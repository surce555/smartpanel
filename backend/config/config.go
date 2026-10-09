package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
)

type Config struct {
	Port          string
	DataDir       string
	UploadsDir    string
	DBPath        string
	JWTSecret     []byte
	EncryptionKey []byte
	DockerSocket  string
}

var AppConfig *Config

func InitConfig() *Config {
	port := os.Getenv("PANEL_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "5666"
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	uploadsDir := os.Getenv("UPLOADS_DIR")
	if uploadsDir == "" {
		uploadsDir = "./uploads"
	}

	jwtSecretStr := os.Getenv("JWT_SECRET")
	if jwtSecretStr == "" {
		jwtSecretStr = "smartpanel-secret-key-nas-dashboard-change-me"
	}

	encKeyStr := os.Getenv("ENCRYPTION_KEY")
	if encKeyStr == "" {
		encKeyStr = "smartpanel-default-32-byte-aes-k!"
	}

	// Derive 32-byte key using sha256 to guarantee 256-bit AES key size
	hash := sha256.Sum256([]byte(encKeyStr))
	encKey := hash[:]

	dockerSocket := os.Getenv("DOCKER_HOST")
	if dockerSocket == "" {
		dockerSocket = "/var/run/docker.sock"
	}

	// Ensure directories exist
	_ = os.MkdirAll(dataDir, 0755)
	_ = os.MkdirAll(uploadsDir, 0755)
	_ = os.MkdirAll(filepath.Join(uploadsDir, "icons"), 0755)
	_ = os.MkdirAll(filepath.Join(uploadsDir, "wallpapers"), 0755)
	_ = os.MkdirAll(filepath.Join(uploadsDir, "fonts"), 0755)
	_ = os.MkdirAll(filepath.Join(dataDir, "logs"), 0755)

	AppConfig = &Config{
		Port:          port,
		DataDir:       dataDir,
		UploadsDir:    uploadsDir,
		DBPath:        filepath.Join(dataDir, "smartpanel.db"),
		JWTSecret:     []byte(jwtSecretStr),
		EncryptionKey: encKey,
		DockerSocket:  dockerSocket,
	}

	return AppConfig
}

// Encrypt encrypts plain text using AES-GCM
func Encrypt(plainText string) (string, error) {
	if plainText == "" {
		return "", nil
	}
	block, err := aes.NewCipher(AppConfig.EncryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plainText), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64 encoded AES-GCM ciphertext
func Decrypt(cipherTextB64 string) (string, error) {
	if cipherTextB64 == "" {
		return "", nil
	}
	data, err := base64.StdEncoding.DecodeString(cipherTextB64)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(AppConfig.EncryptionKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, cipherbytes := data[:nonceSize], data[nonceSize:]
	plain, err := gcm.Open(nil, nonce, cipherbytes, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
