package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
	gin_cors "github.com/rs/cors/wrapper/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Conexão com o BD
var DB *gorm.DB

// Monitor representa um serviço a ser monitorado
type Monitor struct {
	ID            int    `json:"id" gorm:"primaryKey"`
	Name          string `json:"name"`
	URL           string `json:"url"`
	Interval      int    `json:"interval"`       // em segundos
	CurrentStatus string `json:"current_status"` // "Online" or "Offline"
}

// UptimeLog representa o resultado de uma única checagem de saúde
type UptimeLog struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	MonitorID  int       `json:"monitor_id" gorm:"index"`
	Latency    int64     `json:"latency_ms"` // em milissegundos
	StatusCode int       `json:"status_code"`
	Timestamp  time.Time `json:"timestamp"`
}

func initDB() {
	var err error
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=rondon_admin password=rondon_pass123 dbname=uptime_monitor_db port=5432 sslmode=disable"
	}

	for i := 1; i <= 5; i++ {
		DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		fmt.Printf("⚠️ Tentativa %d de conectar ao PostgreSQL falhou. Retentando...\n", i)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		panic("❌ Falha crítica: não foi possível conectar ao PostgreSQL!")
	}

	DB.AutoMigrate(&Monitor{}, &UptimeLog{})
	fmt.Println("📦 Banco de dados PostgreSQL conectado e migrado com sucesso.")
}

func seedMonitors() {
	fitnessUrl := os.Getenv("URL_FITNESS_API")
	if fitnessUrl == "" {
		fitnessUrl = "http://localhost:8000/docs"
	}

	financeUrl := os.Getenv("URL_FINANCE_API")
	if financeUrl == "" {
		financeUrl = "http://localhost:5074/health"
	}

	frontendUrl := os.Getenv("URL_FRONTEND")
	if frontendUrl == "" {
		frontendUrl = "http://localhost:4200"
	}

	var count int64
	DB.Model(&Monitor{}).Count(&count)

	if count == 0 {
		fmt.Println("🌱 Semeando dados iniciais de monitoramento...")
		baseMonitors := []Monitor{
			{Name: "Fitness API (Python)", URL: fitnessUrl, Interval: 10, CurrentStatus: "Pending"},
			{Name: "Finance API (.NET)", URL: financeUrl, Interval: 10, CurrentStatus: "Pending"},
			{Name: "Angular Frontend", URL: frontendUrl, Interval: 10, CurrentStatus: "Pending"},
		}
		DB.Create(&baseMonitors)
	} else {
		DB.Model(&Monitor{}).Where("name = ?", "Fitness API (Python)").Update("url", fitnessUrl)
		DB.Model(&Monitor{}).Where("name = ?", "Finance API (.NET)").Update("url", financeUrl)
		DB.Model(&Monitor{}).Where("name = ?", "Angular Frontend").Update("url", frontendUrl)
		fmt.Printf("✅ URLs sincronizadas: Fitness=%s, Finance=%s\n", fitnessUrl, financeUrl)
	}
}

func main() {
	fmt.Println("🚀 Iniciando Uptime Monitor Engine (Go)...")

	initDB()
	seedMonitors()

	go startCheckingEngine()

	r := gin.Default()

	// AJUSTE DE CORS: Sincronia Total (Railway + Produção Fixa)
	frontendURLEnv := strings.TrimRight(os.Getenv("URL_FRONTEND"), "/")
	if frontendURLEnv == "" {
		frontendURLEnv = "http://localhost:4200"
	}

	r.Use(gin_cors.New(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:4200",
			"https://guilhermerondon-interface.vercel.app", // URL de produção confirmada no console
			frontendURLEnv,
			frontendURLEnv + "/",
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.GET("/api/monitors", func(c *gin.Context) {
		fmt.Println("📡 Monitor Watchdog: Rota /api/monitors acessada.")
		var currentMonitors []Monitor
		DB.Find(&currentMonitors)
		c.JSON(http.StatusOK, currentMonitors)
	})

	r.GET("/api/monitors/:id/history", func(c *gin.Context) {
		id := c.Param("id")
		var history []UptimeLog
		DB.Where("monitor_id = ?", id).Order("timestamp desc").Limit(24).Find(&history)
		c.JSON(http.StatusOK, history)
	})
	
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "Healthy", "service": "Uptime Watchdog"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("📡 Servidor da API rodando na porta %s\n", port)
	r.Run(":" + port)
}

func startCheckingEngine() {
	for {
		var wg sync.WaitGroup
		var activeMonitors []Monitor
		DB.Find(&activeMonitors)

		for i := range activeMonitors {
			wg.Add(1)
			go checkMonitor(&activeMonitors[i], &wg)
		}

		wg.Wait()
		time.Sleep(15 * time.Second)
	}
}

func checkMonitor(m *Monitor, wg *sync.WaitGroup) {
	defer wg.Done()

	type checkResult struct {
		statusCode int
		latency    int64
		err        error
	}
	resultChan := make(chan checkResult, 1)

	go func() {
		start := time.Now()
		resp, err := http.Get(m.URL)
		latency := time.Since(start).Milliseconds()

		if err != nil {
			resultChan <- checkResult{0, latency, err}
			return
		}
		defer resp.Body.Close()
		resultChan <- checkResult{resp.StatusCode, latency, nil}
	}()

	select {
	case res := <-resultChan:
		isOnline := res.statusCode == 200 || (res.statusCode == 401 && strings.Contains(m.URL, "vercel.app"))

		if res.err != nil || !isOnline {
			m.CurrentStatus = "Offline"
		} else {
			m.CurrentStatus = "Online"
		}
		DB.Save(m)
		saveLog(m.ID, res.latency, res.statusCode)

	case <-time.After(8 * time.Second):
		m.CurrentStatus = "Offline"
		DB.Save(m)
		saveLog(m.ID, 8000, 408)
	}
}

func saveLog(monitorID int, latency int64, statusCode int) {
	newLog := UptimeLog{
		MonitorID:  monitorID,
		Latency:    latency,
		StatusCode: statusCode,
		Timestamp:  time.Now(),
	}
	DB.Create(&newLog)
}
