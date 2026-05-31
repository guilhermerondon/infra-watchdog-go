package main

import (
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Permite conexões do Angular (CORS liberado para o monitor)
	},
}

type WebSocketServer struct {
	clients map[*websocket.Conn]bool
	mu      sync.Mutex
}

func NewWebSocketServer() *WebSocketServer {
	return &WebSocketServer{
		clients: make(map[*websocket.Conn]bool),
	}
}

// Rota que o Angular vai chamar para abrir o túnel
func (ws *WebSocketServer) HandleConnections(w http.ResponseWriter, r *http.Request) {
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Erro ao estabelecer WebSocket: %v", err)
		return
	}

	ws.mu.Lock()
	ws.clients[wsConn] = true
	ws.mu.Unlock()
	log.Println("Novo cliente conectado ao painel de monitoramento!")

	// Mantém a conexão aberta lendo mensagens (mesmo que descartemos)
	go func() {
		defer func() {
			ws.mu.Lock()
			delete(ws.clients, wsConn)
			ws.mu.Unlock()
			wsConn.Close()
		}()
		for {
			if _, _, err := wsConn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

// Função para o Go disparar atualizações para o frontend
func (ws *WebSocketServer) BroadcastStatus(statusPayload interface{}) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	for client := range ws.clients {
		err := client.WriteJSON(statusPayload)
		if err != nil {
			log.Printf("Erro ao enviar mensagem, desconectando cliente: %v", err)
			client.Close()
			delete(ws.clients, client)
		}
	}
}
