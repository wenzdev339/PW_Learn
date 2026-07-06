// Package config รวบรวมค่า config ทั้งหมดของแอปไว้ที่เดียว (อ่านจาก
// environment variable) เพื่อไม่ให้ต้องเรียก os.Getenv กระจายอยู่ทั่วโค้ด —
// ทุกที่ที่ต้องใช้ config รับ Config struct เป็น parameter แทน
package config

import (
	"os"
	"strconv"
)

type Config struct {
	AppEnv                  string
	Port                    string
	DatabaseURL             string
	JWTAccessSecret         string
	JWTRefreshSecret        string
	AllowTestReset          bool
	SimulateLatencyMs       int
	SimulateLatencyJitterMs int
	SimulateErrorRate       float64
	SimulateRateLimit       int
}

// Load reads configuration from OS environment variables. Call
// godotenv.Load() before Load() if you want a local .env file merged in.
func Load() Config {
	return Config{
		AppEnv:                  getEnv("APP_ENV", "development"),
		Port:                    getEnv("PORT", "4000"),
		DatabaseURL:             mustGetEnv("DATABASE_URL"),
		JWTAccessSecret:         mustGetEnv("JWT_ACCESS_SECRET"),
		JWTRefreshSecret:        mustGetEnv("JWT_REFRESH_SECRET"),
		AllowTestReset:          getEnvBool("ALLOW_TEST_RESET", false),
		SimulateLatencyMs:       getEnvInt("SIMULATE_LATENCY_MS", 0),
		SimulateLatencyJitterMs: getEnvInt("SIMULATE_LATENCY_JITTER_MS", 0),
		SimulateErrorRate:       getEnvFloat("SIMULATE_ERROR_RATE", 0),
		SimulateRateLimit:       getEnvInt("SIMULATE_RATE_LIMIT", 1000),
	}
}

// TestConfig คืนค่า config คงที่สำหรับใช้ในเทส Go (go test) — ไม่พึ่งไฟล์ .env
// หรือ working directory เลย เพราะ `go test` รันแต่ละ package จาก directory
// ของตัวเอง ทำให้ path แบบ relative ไปยัง .env ไม่เสถียร การ hardcode ค่าที่นี่
// จึงทำให้เทสรันได้เหมือนกันทุกเครื่อง สามารถ override connection string ผ่าน
// env var TEST_DATABASE_URL ได้ (เช่น ตอนรันบน CI ที่ host ต่างจาก localhost)
func TestConfig() Config {
	return Config{
		AppEnv:                  "test",
		Port:                    "4001",
		DatabaseURL:             getEnv("TEST_DATABASE_URL", "postgres://postgres:postgres@localhost:5433/pwlearn_test?sslmode=disable"),
		JWTAccessSecret:         "test-access-secret",
		JWTRefreshSecret:        "test-refresh-secret",
		AllowTestReset:          true,
		SimulateLatencyMs:       0,
		SimulateLatencyJitterMs: 0,
		SimulateErrorRate:       0,
		SimulateRateLimit:       1000,
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func mustGetEnv(key string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		panic("missing required env var: " + key)
	}
	return v
}

func getEnvBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvFloat(key string, fallback float64) float64 {
	v, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}
