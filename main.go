package main

import (
	"fmt"
	"net/http"
	"os"
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
	ID            int       `json:"id" gorm:"primaryKey"`
	Name          string    `json:"name"`
	URL           string    `json:"url"`
	Interval      int       `json:"interval"` // em segundos
	CurrentStatus string    `json:"current_status"` // "Online" or "Offline"
}

// UptimeLog representa o resultado de uma única checagem de saúde
type UptimeLog struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	MonitorID  int       `json:"monitor_id" gorm:"index"`
	Latency    int64     `json:"latency_ms"` // em milissegundos
	StatusCode int       `json:"status_code"`
	Timestamp  time.Time `json:"timestamp"`
}

// Lista global removida, utilizando BD agora

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
		fmt.Printf("⚠️ Tentativa %d de conectar ao PostgreSQL falhou. Retentando em 2 segundos...\n", i)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		panic("❌ Falha crítica: não foi possível conectar ao PostgreSQL após 5 tentativas!")
	}

	// Migrar o esquema
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
		financeUrl = "http://localhost:5074/api/Transactions"
	}
	frontendUrl := os.Getenv("URL_FRONTEND")
	if frontendUrl == "" {
		frontendUrl = "http://localhost:4200"
	}

	var count int64
	DB.Model(&Monitor{}).Count(&count)
	
	if count == 0 {
		fmt.Println("🌱 Tabela de monitores vazia. Semeando dados iniciais...")
		baseMonitors := []Monitor{
			{Name: "Fitness API (Python)", URL: fitnessUrl, Interval: 10, CurrentStatus: "Pending"},
			{Name: "Finance API (.NET)", URL: financeUrl, Interval: 10, CurrentStatus: "Pending"},
			{Name: "Angular Frontend", URL: frontendUrl, Interval: 10, CurrentStatus: "Pending"},
		}
		DB.Create(&baseMonitors)
		fmt.Println("✅ 3 Monitores base criados com sucesso!")
	} else {
		// Atualiza as URLs existentes caso as variáveis de ambiente mudem
		DB.Model(&Monitor{}).Where("name = ?", "Fitness API (Python)").Update("url", fitnessUrl)
		DB.Model(&Monitor{}).Where("name = ?", "Finance API (.NET)").Update("url", financeUrl)
		DB.Model(&Monitor{}).Where("name = ?", "Angular Frontend").Update("url", frontendUrl)
		fmt.Println("✅ URLs de monitores sincronizadas com o ambiente atual.")
	}
}

func main() {
	fmt.Println("🚀 Iniciando Uptime Monitor Engine (Go)...")

	initDB()
	seedMonitors()

	// Inicia o motor de checagem em uma goroutine rodando em background
	go startCheckingEngine()

	// Configuração do Router (Gin Framework)
	r := gin.Default()
	
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:4200"
	}

	// Middleware de CORS usando rs/cors
	r.Use(gin_cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:4200", frontendURL},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	}))

	// Rota para listar todos os monitores e seu status atual
	r.GET("/api/monitors", func(c *gin.Context) {
		var currentMonitors []Monitor
		DB.Find(&currentMonitors)
		c.JSON(http.StatusOK, currentMonitors)
	})

	// Rota para ver os logs históricos de um monitor (últimos 24)
	r.GET("/api/monitors/:id/history", func(c *gin.Context) {
		id := c.Param("id")
		var history []UptimeLog
		DB.Where("monitor_id = ?", id).Order("timestamp desc").Limit(24).Find(&history)
		c.JSON(http.StatusOK, history)
	})

	// Rota auxiliar para ver os logs recentes (opcional agora que temos histórico)
	r.GET("/api/logs", func(c *gin.Context) {
		var recentLogs []UptimeLog
		DB.Order("timestamp desc").Limit(50).Find(&recentLogs)
		c.JSON(http.StatusOK, recentLogs)
	})

	// Inicia o servidor na porta 8080
	fmt.Println("📡 Servidor da API rodando na porta 8080")
	r.Run(":8080")
}

// startCheckingEngine itera sobre todos os monitores e dispara uma goroutine para cada um
func startCheckingEngine() {
	for {
		var wg sync.WaitGroup
		var activeMonitors []Monitor
		DB.Find(&activeMonitors)
		
		for i := range activeMonitors {
			wg.Add(1) // Adiciona 1 ao WaitGroup para cada monitor
			go checkMonitor(&activeMonitors[i], &wg) // Execução simultânea
		}
		
		// Aguarda até que todas as verificações deste ciclo terminem
		wg.Wait()
		
		// Aguarda o intervalo antes do próximo ciclo
		time.Sleep(10 * time.Second)
	}
}

// checkMonitor realiza uma requisição HTTP para a URL do monitor usando Timeout com Select
func checkMonitor(m *Monitor, wg *sync.WaitGroup) {
	defer wg.Done() // Sinaliza que esta goroutine terminou ao final

	// Canal para receber o resultado da checagem
	type checkResult struct {
		statusCode int
		latency    int64
		err        error
	}
	resultChan := make(chan checkResult, 1)

	// Realiza a requisição numa goroutine separada
	go func() {
		start := time.Now()
		
		// Fazendo GET simples na URL
		resp, err := http.Get(m.URL)
		latency := time.Since(start).Milliseconds()
		
		if err != nil {
			resultChan <- checkResult{0, latency, err}
			return
		}
		defer resp.Body.Close()
		
		resultChan <- checkResult{resp.StatusCode, latency, nil}
	}()

	// Select aguarda a resposta do canal OU um timeout estourar
	select {
	case res := <-resultChan:
		// Regra de Negócio Refinada: Apenas status 200 OK significa (ONLINE)
		if res.err != nil || res.statusCode != 200 {
			m.CurrentStatus = "Offline"
			DB.Save(m)
			saveLog(m.ID, res.latency, res.statusCode)
			fmt.Printf("❌ [OFFLINE] %s (%s) - Erro/Status: %d\n", m.Name, m.URL, res.statusCode)
		} else {
			m.CurrentStatus = "Online"
			DB.Save(m)
			saveLog(m.ID, res.latency, res.statusCode)
			fmt.Printf("✅ [ONLINE] %s (%s) - %dms\n", m.Name, m.URL, res.latency)
		}
	case <-time.After(5 * time.Second):
		// Timeout customizado disparado
		m.CurrentStatus = "Offline"
		DB.Save(m)
		saveLog(m.ID, 5000, 408) // 408 Request Timeout
		fmt.Printf("⚠️ [TIMEOUT] %s (%s) demorou mais de 5s para responder\n", m.Name, m.URL)
	}
}

// saveLog persiste o resultado da checagem no SQLite
func saveLog(monitorID int, latency int64, statusCode int) {
	newLog := UptimeLog{
		MonitorID:  monitorID,
		Latency:    latency,
		StatusCode: statusCode,
		Timestamp:  time.Now(),
	}
	DB.Create(&newLog)
}
