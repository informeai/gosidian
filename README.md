# gosidian

Um aplicativo de notas de terminal estilo obsidian, construído com Bubble Tea.

## Recursos

- Interface de terminal estilizada
- Lista de notas com navegação
- Editor de texto multi-linha
- Linkagem de notas com `[[nome-da-nota]]`
- Atalhos de teclado
- Notas salvas em Markdown

## Instalação

```bash
go install github.com/informeai/gosidian@latest
```

## Uso

```bash
gosidian
```

## Atalhos

### Lista de Notas

| Tecla | Ação |
|------|------|
| `↑` / `k` | Move cursor para cima |
| `↓` / `j` | Move cursor para baixo |
| `n` | Nova nota |
| `o` / `Enter` | Abrir nota selecionada |
| `q` | Sair |

### Editor

| Tecla | Ação |
|------|------|
| `Ctrl+S` | Salvar nota |
| `Ctrl+R` | Renomear nota |
| `Ctrl+L` | Autocomplete de links |
| `Esc` | Voltar para lista |

### Visualização

| Tecla | Ação |
|------|------|
| `Enter` | Ir para nota linkada |
| `Alt+←` | Voltar no histórico |
| `Alt+→` | Avançar no histórico |
| `e` | Editar nota |

## Armazenamento

As notas são salvas em `~/.gosidian/` no formato Markdown.