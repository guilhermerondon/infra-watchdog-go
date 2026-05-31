package main

import (
	"log"
	"net/http"
	"time"
)

// Estrutura para mapear os serviços que vamos monitorar
type ServiceTarget struct {
	Name string
	URL  string
}

func StartHealthChecks(ws *WebSocketServer) {
	// Configura um cliente HTTP com timeout estrito de 2 segundos.
	// Se o serviço demorar mais que isso, consideramos fora do ar.
	client := http.Client{
		Timeout: 2 * time.Second,
	}

	// Defina os endpoints reais aqui (ajuste as URLs conforme o seu ambiente)
	targets := []ServiceTarget{
		{Name: "Athlete API (Python)", URL: "https://fitness-api-free.onrender.com/docs"}, // Endpoint que retorna 200 OK
		{Name: "Finance API (.NET)", URL: "http://localhost:5000/health"},                 // Ajuste para a URL real da sua API C#
	}

	// Loop infinito rodando em background
	for {
		for _, target := range targets {
			go func(t ServiceTarget) {
				start := time.Now()
				resp, err := client.Get(t.URL)
				latency := time.Since(start).Milliseconds()

				status := "online"

				// Lógica de degradação e falha
				if err != nil {
					status = "offline"
					latency = 0
					log.Printf("[!] Falha de conexão no %s: %v", t.Name, err)
				} else {
					defer resp.Body.Close()
					if resp.StatusCode >= 400 {
						status = "offline"
					} else if latency > 1000 {
						// Se demorar mais de 1 segundo para responder, está degradado
						status = "degraded"
					}
				}

				// Dispara o status real para o Angular
				ws.BroadcastStatus(map[string]interface{}{
					"service":    t.Name,
					"status":     status,
					"latency_ms": latency,
				})
			}(target)
		}
		// Aguarda 5 segundos antes da próxima rodada de pings
		time.Sleep(5 * time.Second)
	}
}
