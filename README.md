# Frogger Concorrente — FPPD T1 Opção B

Jogo Frogger de terminal em Go com arquitetura concorrente baseada em goroutines e channels. Trabalho da disciplina **FPPD 98713-04** (PUCRS).

## Grupo

- Ana Clara — [matrícula]
- [Integrante 2] — [matrícula]

## Como executar

```bash
go run .
go run -race .
```

## Controles

W/A/S/D para mover, Q para sair.

## Arquitetura

7 goroutines, comunicação por channels, shutdown gracioso via context. Detalhes em `docs/arquitetura.pdf`.
