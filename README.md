# gosidian

Um aplicativo de notas de terminal estilo obsidian, construído com Bubble Tea.

## Recursos

- Interface de terminal estilizada
- Lista de notas com navegação
- Editor de texto multi-linha
- Linkagem de notas com `[[nome-da-nota]]`
- Autocomplete ao digitar `[[`
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
| `Ctrl+L` | Iniciar/linkar nota |
| `Tab` / `↓` | Próxima sugestão |
| `Shift+Tab` / `↑` | Sugestão anterior |
| `Enter` | Inserir link selecionado |
| `Esc` | Voltar para lista |

### Linkagem

Digite `[[` e o autocomplete mostrará as notas existentes. Use `Tab`/`Enter` para selecionar.

Exemplo: `[[minha-nota]]`

## Armazenamento

As notas são salvas em `~/.gosidian/` no formato Markdown.