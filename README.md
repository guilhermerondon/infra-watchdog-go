<img width="100%" src="https://capsule-render.vercel.app/api?type=waving&color=8b5cf6&height=110&section=header&animation=fadeIn"/>

# Infrastructure Pulse / Watchdog (Go)

Engine assíncrona de alta performance desenvolvida em Go, responsável pelo monitoramento contínuo, cálculo de latência e checagem de integridade (*Uptime & Health Checks*) dos microsserviços do ecossistema. O sistema opera de forma concorrente e não-bloqueante, expondo métricas em tempo real para alimentação do frontend.

---

## 🚀 Tecnologias e Arquitetura

* **Go (Golang)**: Escolhido estrategicamente devido à sua compilação nativa, baixo consumo de memória e velocidade na execução de rotinas de I/O de rede.
* **Gin Framework**: Roteamento HTTP de altíssima performance estruturado sobre uma árvore de caminhos de rádio (*Radix Tree*), garantindo tempo de resposta mínimo nos endpoints de telemetria.
* **Concorrência Nativa (Goroutines & Channels)**: Pooling de monitoramento construído com rotinas leves assíncronas acionadas por um `time.Ticker`. Os resultados das requisições paralelas são orquestrados e sincronizados via canais (*Channels*) e estruturas de controle do pacote `sync`.
* **GORM & PostgreSQL**: Abstração de persistência otimizada para o armazenamento histórico de séries temporais de latência, códigos de status HTTP e registros de oscilação de infraestrutura (*downtimes*).

---

## 🛠️ Engenharia de Monitoramento (Como Funciona)

O motor do Watchdog consome uma lista dinâmica de endpoints (incluindo o backend em .NET Core e a API FastAPI em Python). A cada ciclo determinado, uma Goroutine dedicada é disparada para efetuar um handshake HTTP em paralelo. Se um serviço falhar ou demorar mais do que o limite estipulado (*timeout*), o motor atualiza o estado interno e emite o pulso de alerta que é captado instantaneamente no painel web.

---

## ⚙️ Execução Local

### Pré-requisitos
* Go v1.21+ instalado.
* Instância do PostgreSQL ativa.

### Instalação e Inicialização
```bash
# Limpar e sincronizar as dependências e módulos do projeto
go mod tidy

# Compilar e rodar o servidor HTTP e o motor de pooling
go run main.go
