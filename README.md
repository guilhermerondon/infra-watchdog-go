# Infra Watchdog Go

Sistema de monitoramento contínuo (Uptime) focado em resiliência e concorrência, responsável por garantir que as outras APIs do ecossistema estejam sempre ativas.

## 🚀 Tecnologias e Arquitetura

- **Go (Golang)**: Linguagem compilada, focada em performance e simplicidade.
- **Gin Framework**: Roteamento HTTP extremamente rápido.
- **Goroutines & Canais**: Verificações concorrentes de status usando o poder do Go nativo.
- **GORM (PostgreSQL)**: Registro histórico de latência e _status codes_.

## 🧠 Filosofia

Monitoramento constante é como a consistência de um treino de 6 vezes por semana: sem falhar, sempre presente. O Watchdog trabalha nos bastidores, checando ativamente os sinais vitais do ecossistema para que a interface reflita o estado real, 24/7.

## 🛠️ Como Executar

```bash
go mod tidy
go run main.go
```
O serviço expõe métricas para consumo do frontend em tempo real.
