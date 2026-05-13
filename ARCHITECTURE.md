# Arquitetura concorrente do Frogger (resumo)

Este documento descreve a arquitetura de goroutines e channels usada no projeto, com explicações simples para estudantes.

## 1) Visão geral

O jogo é organizado em várias goroutines com responsabilidades distintas que se comunicam por channels (sem memória compartilhada não protegida):

- Input reader: lê teclado e envia `Command` para `inputCh`.
- Ticker: gera "ticks" de tempo no canal `tickCh` (30 FPS).
- Lanes: uma goroutine por faixa (lane) que move carros e envia `CarEvent` para `carEventsCh`.
- Game loop: recebe `tickCh`, `inputCh`, `carEventsCh` e mantém o `GameState` principal; envia snapshots para `renderCh`.
- Renderer: consome `GameState` (snapshots) e desenha no terminal.

Todos os componentes observam um `context.Context` passado a partir de `main`, que permite shutdown gracioso.

## 2) Diagrama de goroutines e channels (texto)

Goroutines:

- main
  - cria: inputReader (go), ticker (go), lane-2 (go), lane-5 (go), lane-7 (go), renderer (go), gameLoop (go)

Channels e fluxo (quem envia → quem recebe):

- inputCh (chan Command): inputReader → gameLoop
- tickCh (chan struct{}): ticker → gameLoop
- carEventsCh (chan CarEvent): lane-* → gameLoop
- renderCh (chan GameState): gameLoop → renderer
- sigCh (chan os.Signal): os/signal → main (para cancelar o context)

Fluxo de dados (linha a linha):

inputReader -> inputCh -> gameLoop -> renderCh -> renderer
ticker -> tickCh -> gameLoop
lane-* -> carEventsCh -> gameLoop

## 3) Contrato das principais goroutines (entrada/saída/erros)

- inputReader
  - Inputs: stdin (bytes)
  - Outputs: `Command` via `inputCh`
  - Erros: se o stdin for fechado, encerra.
  - Sucesso: envia comandos enquanto o contexto não for cancelado.

- ticker
  - Inputs: tempo
  - Outputs: evento vazio no `tickCh` a 30 FPS (envio não bloqueante)
  - Erros: nenhum

- lane (por faixa)
  - Inputs: timer por faixa
  - Outputs: `CarEvent` com snapshot de []Car
  - Erros: nenhum

- gameLoop
  - Inputs: `inputCh`, `tickCh`, `carEventsCh`
  - Outputs: snapshots imutáveis `GameState` em `renderCh`
  - Erros: chama `cancel()` para encerrar o sistema quando o jogo termina

- renderer
  - Inputs: `GameState` snapshots
  - Outputs: escrita em stdout

## 4) Como o shutdown gracioso funciona

- `main` cria um `context.WithCancel` e passa o `ctx` para todas as goroutines.
- Ao receber SIGINT/SIGTERM, `main` chama `cancel()`.
- Cada goroutine faz `select` em `<-ctx.Done()` e encerra quando detecta cancelamento.
- O `WaitGroup` é usado por `main` para aguardar todas as goroutines terminarem.
- O `gameLoop` também pode chamar `cancel()` internamente (ex.: fim do jogo ou CmdQuit).

Isso garante que não haja goroutines vazando: todas observam o mesmo contexto e encerram.

## 5) Notas sobre segurança contra data races

- Comunicação entre goroutines é feita por channels; o estado mutável principal (`GameState`) fica somente no game loop.
- Antes de enviar para o renderer, o game loop faz uma cópia profunda (deep copy) das slices de carros, garantindo que o renderer tenha uma visão independente e não cause data races.
- As lanes enviam snapshots (cópias) de seus slices ao invés de compartilhar o slice original.

## 6) Requisitos cobridos

- Goroutines distintas: ticker, inputReader, lanes (3), gameLoop, renderer → >4 goroutines com papéis distintos.
- Channels: `inputCh`, `tickCh`, `carEventsCh`, `renderCh`, `sigCh` usados para comunicação.
- `select`: usado no gameLoop para multiplexar `inputCh`, `tickCh` e `carEventsCh`.
- Entidades autônomas: lanes são entidades independentes que movem carros.
- Shutdown gracioso: `context` com cancelamento passado a todas as goroutines.
- Data race: evitado com ownership do state no gameLoop e deep copies antes de envio.
