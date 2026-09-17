package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/emersonjoe/trilha-spec/spec"
)

// lang is the language the CLI speaks, read once from TRILHA_LANG: `pt` or
// `pt-BR` (any case, `_` or `-`) select Portuguese; anything else, including
// unset, is English. File formats and --json never change with it: status
// names, field names and IDs are protocol, not messages.
var lang = detectLang(os.Getenv("TRILHA_LANG"))

func detectLang(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "pt" || strings.HasPrefix(v, "pt-") || strings.HasPrefix(v, "pt_") {
		return "pt"
	}
	return "en"
}

// catalogues holds one message table per language, keyed by the English
// text. English needs no table: the key is the message.
var catalogues = map[string]map[string]string{"pt": pt}

// T translates one CLI message. A key the language does not have falls back
// to English rather than failing: a missing translation is a gap, not a bug
// the operator should hit.
func T(key string) string {
	if c := catalogues[lang]; c != nil {
		if v, ok := c[key]; ok {
			return v
		}
	}
	return key
}

// doctorMessage renders one layout problem in the CLI's language. The codes
// are spec.Problem's; an unknown code falls back to the problem's own text.
func doctorMessage(p spec.Problem) string {
	switch p.Code {
	case spec.ProblemMissingDir:
		return fmt.Sprintf(T("missing directory %s (run `trilha-spec init`)"), p.Arg)
	case spec.ProblemMissingFile:
		return fmt.Sprintf(T("missing %s (run `trilha-spec init`)"), p.Arg)
	case spec.ProblemGitignoreAll:
		return T(".trilha/.gitignore ignores everything (`*`): specs, tasks and evidence will not be committed; run `trilha-spec init` to rewrite it")
	case spec.ProblemSpecRefMissing:
		return fmt.Sprintf(T("spec reference does not exist: %s"), p.Arg)
	case spec.ProblemSpecNoSuccessor:
		return fmt.Sprintf(T("spec %s is superseded but no spec names it in `supersedes`"), p.Arg)
	}
	return p.String()
}

const usagePT = `trilha-spec ` + version + ` — o protocolo aberto para trabalho que agentes executam

uso: trilha-spec <comando> [flags]

  init [dir]                    cria .trilha/ (projeto, constituição, agentes)
  spec new <título> [--issue N] [--body TEXTO | --body-file CAMINHO]
  spec list [--status S] | show <id> | move <id> <status> | set <id> [--issue N] [--supersedes A,B] [--depends A,B]
  task add <título> [--spec ID] [--depends A,B] [--agent N] [--status S] [--accept C]... [--check CMD]...
           [--body TEXTO | --body-file CAMINHO]   (CAMINHO "-" lê stdin)
  task list [--status S] | show <id> | next | move <id> <status> | graph [--dot]
  agent list | show <nome>
  context <task-id>             o pacote de contexto que um agente recebe (--json para ferramentas)
  verify <task-id> [--dir D]    roda os checks da task e grava evidência
  evidence <task-id> [add --note TEXTO]
  mcp [--write]                 serve o protocolo por MCP em stdio
  doctor                        o que um leitor tropeçaria
  version

Toda listagem aceita --json. Status de task: idea spec ready running verify review done blocked failed.
Status de spec: draft approved done rejected superseded.
TRILHA_LANG=pt traduz as mensagens e os templates que init e spec new escrevem; formatos de arquivo e --json não mudam.
`

// pt is the Portuguese (Brazil) table. Keys are the exact English strings
// the code passes to T; keep them in sync when a message changes.
var pt = map[string]string{
	usage:                    usagePT,
	"error:":                 "erro:",
	"unknown command %q\n%s": "comando desconhecido %q\n%s",
	// init
	"%s already initialized; nothing written\n": "%s já inicializado; nada escrito\n",
	"  created %s\n": "  criado %s\n",
	"next: describe the project in .trilha/project.md, then `trilha-spec spec new \"<title>\"`\n": "próximo passo: descreva o projeto em .trilha/project.md e depois `trilha-spec spec new \"<título>\"`\n",
	// spec
	"usage: trilha-spec spec new <title> | list | show <id> | move <id> <status> | set <id> [flags]": "uso: trilha-spec spec new <título> | list | show <id> | move <id> <status> | set <id> [flags]",
	"usage: trilha-spec spec move <id> <status>":                                                     "uso: trilha-spec spec move <id> <status>",
	"usage: trilha-spec spec set <id> [--issue N] [--supersedes A,B] [--depends A,B]":                "uso: trilha-spec spec set <id> [--issue N] [--supersedes A,B] [--depends A,B]",
	"updated %s\n":                         "atualizado %s\n",
	"usage: trilha-spec spec new <title>":  "uso: trilha-spec spec new <título>",
	"usage: trilha-spec spec show <id>":    "uso: trilha-spec spec show <id>",
	"created %s (%s)\n":                    "criado %s (%s)\n",
	"unknown spec command %q":              "subcomando de spec desconhecido %q",
	"--body and --body-file are exclusive": "--body e --body-file são exclusivos",
	// task
	"usage: trilha-spec task add|list|show|next|move|graph": "uso: trilha-spec task add|list|show|next|move|graph",
	"usage: trilha-spec task add <title> [flags]":           "uso: trilha-spec task add <título> [flags]",
	"usage: trilha-spec task show <id>":                     "uso: trilha-spec task show <id>",
	"usage: trilha-spec task move <id> <status>":            "uso: trilha-spec task move <id> <status>",
	"ID":         "ID",
	"STATUS":     "STATUS",
	"TITLE":      "TÍTULO",
	"WAITING ON": "AGUARDANDO",
	"nothing ready: no task is `ready` with every dependency done\n": "nada pronto: nenhuma task está `ready` com todas as dependências `done`\n",
	"%s is now %s\n":          "%s agora está %s\n",
	"unknown task command %q": "subcomando de task desconhecido %q",
	// agent
	"usage: trilha-spec agent list | show <name>": "uso: trilha-spec agent list | show <nome>",
	"usage: trilha-spec agent show <name>":        "uso: trilha-spec agent show <nome>",
	"unknown agent command %q":                    "subcomando de agent desconhecido %q",
	// context, verify, evidence
	"usage: trilha-spec context <task-id>":                    "uso: trilha-spec context <task-id>",
	"usage: trilha-spec verify <task-id> [--dir D]":           "uso: trilha-spec verify <task-id> [--dir D]",
	"evidence: %d record(s) in %s\n":                          "evidência: %d registro(s) em %s\n",
	"verification failed":                                     "verificação falhou",
	"usage: trilha-spec evidence <task-id> [add --note TEXT]": "uso: trilha-spec evidence <task-id> [add --note TEXTO]",
	"evidence add needs --note or --file":                     "evidence add precisa de --note ou --file",
	"recorded #%d (%s)\n":                                     "gravado #%d (%s)\n",
	// mcp (stderr)
	"trilha-spec mcp %s · %s\ntools: %s\n":                                              "trilha-spec mcp %s · %s\nferramentas: %s\n",
	"read-only; pass --write to offer trilha_move, trilha_evidence and trilha_verify\n": "somente leitura; passe --write para oferecer trilha_move, trilha_evidence e trilha_verify\n",
	// doctor
	"✓ %s is healthy\n": "✓ %s está saudável\n",
	"%d problem(s)":     "%d problema(s)",
	"missing directory %s (run `trilha-spec init`)":              "falta o diretório %s (rode `trilha-spec init`)",
	"missing %s (run `trilha-spec init`)":                        "falta %s (rode `trilha-spec init`)",
	"spec reference does not exist: %s":                          "referência a spec inexistente: %s",
	"spec %s is superseded but no spec names it in `supersedes`": "a spec %s está superseded mas nenhuma spec a cita em `supersedes`",
	".trilha/.gitignore ignores everything (`*`): specs, tasks and evidence will not be committed; run `trilha-spec init` to rewrite it": ".trilha/.gitignore ignora tudo (`*`): specs, tasks e evidência não serão commitadas; rode `trilha-spec init` para reescrevê-lo",
}
