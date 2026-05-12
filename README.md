# Frogger Concorrente — FPPD T1 Opção B

Jogo Frogger de terminal em Go com arquitetura concorrente baseada em goroutines e channels. Trabalho da disciplina **FPPD 98713-04** (PUCRS).

## Grupo

- Ana Clara Purper da Silva — [24200820]
- Bianca Piassini — [24201030]
- Tarciso Mota — [24200676]
---

## Pré-requisitos

- Go 1.21+ instalado (`go version`)
- Terminal com suporte a ANSI color codes (qualquer terminal moderno no Linux/macOS)

---

## Como executar

```bash
# Clone ou acesse o diretório do projeto
cd fppd-t1-frogger

# Baixar dependências
go mod tidy

# Executar normalmente
go run .

# Executar com detector de corridas (requer terminal interativo)
go run -race .

# Rodar suite de testes com detector de corridas (funciona sem TTY)
go test -race -v ./...
```

> **Nota:** `go run .` e `go run -race .` exigem que o stdin seja um terminal real (TTY).  
> Usar `go test -race ./...` é o método recomendado para CI / ambientes sem terminal.

---

## Controles

| Tecla | Ação       |
|-------|------------|
| W     | Mover para cima |
| S     | Mover para baixo |
| A     | Mover para esquerda |
| D     | Mover para direita |
| Q     | Sair |
| Ctrl+C | Sair (SIGINT) |

---

## Mapa

```
Linha 0  [ objetivo — chegar aqui = vitória ]
Linha 1  [ segura ]
Linha 2  [ faixa de carros → direita ]
Linha 3  [ segura ]
Linha 4  [ segura ]
Linha 5  [ faixa de carros ← esquerda ]
Linha 6  [ segura ]
Linha 7  [ faixa de carros → direita ]
Linha 8  [ segura ]
Linha 9  [ base — posição inicial do sapo ]
```

Grade: **20 colunas × 10 linhas**.

---

## Arquitetura

### 7 Goroutines

| # | Nome | Arquivo | Papel |
|---|------|---------|-------|
| 1 | `runInput` | `input.go` | Lê teclado em raw mode, envia `Command` em `inputCh` |
| 2 | `runTicker` | `gameloop.go` | `time.Ticker` a 30 FPS, envia tick em `tickCh` |
| 3 | `runGameLoop` | `gameloop.go` | Estado autoritativo; multiplexing de todos os channels |
| 4 | `runLane` (linha 2) | `lanes.go` | Spawn/movimento de carros → direita |
| 5 | `runLane` (linha 5) | `lanes.go` | Spawn/movimento de carros ← esquerda |
| 6 | `runLane` (linha 7) | `lanes.go` | Spawn/movimento de carros → direita |
| 7 | `runRenderer` | `renderer.go` | Recebe snapshots, redesenha terminal com ANSI |

### Channels

```
inputCh      chan Command    (buf 8)   — input → gameLoop
tickCh       chan struct{}   (buf 1)   — ticker → gameLoop
carEventsCh  chan CarEvent   (buf 32)  — lanes  → gameLoop
renderCh     chan GameState  (buf 1)   — gameLoop → renderer
```

### Regras de concorrência

- **`GameState` é mutado exclusivamente por `runGameLoop`** — sem mutex.
- Lane goroutines alocam um novo slice a cada tick; o slice antigo torna-se imutável após ser enviado pelo channel (sem race de leitura/escrita).
- Snapshot enviado ao renderer é uma cópia de valor de `GameState` (array `[10][]Car` copiado por valor). Os slices apontam para backing arrays somente-leitura após o envio → sem data race.
- Envios ao `renderCh` são **não-bloqueantes** (`select { case …: default: }`).
- Toda goroutine de longa duração escuta `ctx.Done()` no seu `select`.

### Shutdown gracioso

1. Tecla Q, vitória/derrota, ou SIGINT → `cancel()` chamado.
2. Todas as goroutines detectam `ctx.Done()` e retornam.
3. `wg.Wait()` em `main` aguarda encerramento completo.
4. `term.Restore` restaura o terminal; cursor reativado com `\x1b[?25h`.

---

## Estrutura de arquivos

```
.
├── go.mod
├── go.sum
├── state.go      # tipos: Command, Direction, Car, CarEvent, GameState, LaneDef
├── main.go       # orquestração: context, WaitGroup, channels, signal, goroutines
├── input.go      # goroutine de teclado (raw mode)
├── lanes.go      # goroutines das faixas de trânsito
├── gameloop.go   # ticker + game loop (estado autoritativo)
├── renderer.go   # renderização ANSI
└── main_test.go  # teste de concorrência com -race (sem TTY)
```
