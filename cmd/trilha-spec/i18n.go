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
	case spec.ProblemPrivateKey:
		return fmt.Sprintf(T("private key %s is inside .trilha; move it out (only `.pub` files belong in keys/)"), p.Arg)
	case spec.ProblemSpecNoSecurity:
		return fmt.Sprintf(T("spec %s is approved but declares no security impact (assets, trust_boundaries, controls, evidence)"), p.Arg)
	case spec.ProblemRequirementDuplicate:
		return fmt.Sprintf(T("requirement declared by more than one spec: %s"), p.Arg)
	case spec.ProblemRequirementUnknown:
		return fmt.Sprintf(T("task covers a requirement no spec declares: %s"), p.Arg)
	case spec.ProblemRequirementUncovered:
		return fmt.Sprintf(T("requirement no task covers: %s"), p.Arg)
	case spec.ProblemMilestoneUnknown:
		return fmt.Sprintf(T("milestone project.md does not declare: %s"), p.Arg)
	case spec.ProblemMilestoneEmpty:
		return fmt.Sprintf(T("milestone %s has no task"), p.Arg)
	case spec.ProblemMilestonePastDue:
		return fmt.Sprintf(T("task past its milestone's due date: %s"), p.Arg)
	case spec.ProblemMetricNotEvidenced:
		return fmt.Sprintf(T("acceptance metric with no eval evidence: %s"), p.Arg)
	case spec.ProblemQuorumWithoutRoles:
		return fmt.Sprintf(T("task %s asks for a review quorum but names no roles"), p.Arg)
	case spec.ProblemAttestationUnknownKey:
		return fmt.Sprintf(T("attestation signed with a key the project does not hold: %s"), p.Arg)
	case spec.ProblemRepoUnknown:
		return fmt.Sprintf(T("dependency on a repository project.md does not declare in `repos`: %s"), p.Arg)
	}
	return p.String()
}

const usagePT = `trilha-spec ` + version + ` — o protocolo aberto para trabalho que agentes executam

uso: trilha-spec <comando> [flags]

  init [dir]                    cria .trilha/ (projeto, constituição, agentes)
  spec new <título> [--issue N] [--body TEXTO | --body-file CAMINHO] [--asset A]... [--boundary B]... [--control C]... [--evidence CMD]...
  spec list [--status S] | show <id> [--coverage] | move <id> <status>
  spec set <id> [--issue N] [--supersedes A,B] [--depends A,B] [--asset A]... [--boundary B]... [--control C]... [--evidence CMD]...
  task add <título> [--spec ID] [--depends A,B] [--agent N] [--status S] [--covers R,S] [--milestone M] [--accept C]... [--check CMD]...
           [--body TEXTO | --body-file CAMINHO]   (CAMINHO "-" lê stdin)
  task list [--status S] [--milestone M] | show <id> | next | move <id> <status> | graph [--dot] [--program]
           list, next, graph e doctor aceitam --repo alias=caminho (repetível) para um checkout irmão
  agent list | show <nome>
  project show | pause [--reason R] | resume | limit <chave> <valor|->
  project milestone <id> [--title T] [--due AAAA-MM-DD] [--gate G] | <id> -
  context <task-id>             o pacote de contexto que um agente recebe (--json para ferramentas)
  verify <task-id> [--dir D]    roda os checks da task e grava evidência
  evidence <task-id> [--verify] [--keys DIR]   registros; --verify confere assinaturas contra DIR (padrão .trilha/keys)
  evidence <task-id> add --note TEXTO | add --run [--provider P --model M --tokens-in N --tokens-out N --cost C --currency USD]
  evidence <task-id> add --eval --metric M --value V [--unit U] [--threshold T --comparator >=] [--dataset ID [--dataset-sha256 H]]
  evidence <task-id> add --attestation --by NOME --role R --statement S [--ref X]...   assine, ou não conta para o quórum
           [--sign-key ARQUIVO [--key-id ID]]   assina o registro com uma chave privada Ed25519
  keygen <key-id> [--out DIR]   um par Ed25519: DIR/<key-id>.key (privada, padrão ~/.trilha/keys) e .trilha/keys/<key-id>.pub
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
	"usage: trilha-spec spec new <title> | list | show <id> | move <id> <status> | set <id> [flags]":                                                        "uso: trilha-spec spec new <título> | list | show <id> | move <id> <status> | set <id> [flags]",
	"usage: trilha-spec spec move <id> <status>":                                                                                                            "uso: trilha-spec spec move <id> <status>",
	"usage: trilha-spec spec set <id> [--issue N] [--supersedes A,B] [--depends A,B] [--asset A]... [--boundary B]... [--control C]... [--evidence CMD]...": "uso: trilha-spec spec set <id> [--issue N] [--supersedes A,B] [--depends A,B] [--asset A]... [--boundary B]... [--control C]... [--evidence CMD]...",
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
	"ID":          "ID",
	"STATUS":      "STATUS",
	"TITLE":       "TÍTULO",
	"WAITING ON":  "AGUARDANDO",
	"REQUIREMENT": "REQUISITO",
	"TEXT":        "TEXTO",
	"TASKS":       "TASKS",
	"nothing ready: no task is `ready` with every dependency done\n": "nada pronto: nenhuma task está `ready` com todas as dependências `done`\n",
	"%s is now %s\n":          "%s agora está %s\n",
	"unknown task command %q": "subcomando de task desconhecido %q",
	// agent
	"usage: trilha-spec agent list | show <name>": "uso: trilha-spec agent list | show <nome>",
	"usage: trilha-spec agent show <name>":        "uso: trilha-spec agent show <nome>",
	"unknown agent command %q":                    "subcomando de agent desconhecido %q",
	// project
	"usage: trilha-spec project show | pause [--reason R] | resume | limit <key> <value|-> | milestone <id> [flags]": "uso: trilha-spec project show | pause [--reason R] | resume | limit <chave> <valor|-> | milestone <id> [flags]",
	"usage: trilha-spec project milestone <id> [--title T] [--due YYYY-MM-DD] [--gate G] | <id> -":                   "uso: trilha-spec project milestone <id> [--title T] [--due AAAA-MM-DD] [--gate G] | <id> -",
	"milestone %s is not declared":                     "o marco %s não está declarado",
	"usage: trilha-spec project limit <key> <value|->": "uso: trilha-spec project limit <chave> <valor|->",
	"project is paused: %s\n":                          "projeto pausado: %s\n",
	"%s is paused: %s\n":                               "%s está pausado: %s\n",
	"%s resumed\n":                                     "%s retomado\n",
	"limit %s: %q is not a number":                     "limite %s: %q não é um número",
	"unknown project command %q":                       "subcomando de project desconhecido %q",
	// context, verify, evidence
	"usage: trilha-spec context <task-id>":          "uso: trilha-spec context <task-id>",
	"usage: trilha-spec verify <task-id> [--dir D]": "uso: trilha-spec verify <task-id> [--dir D]",
	"evidence: %d record(s) in %s\n":                "evidência: %d registro(s) em %s\n",
	"verification failed":                           "verificação falhou",
	"usage: trilha-spec evidence <task-id> [add --note TEXT | add --run [--provider P] [--model M] [--tokens-in N] [--tokens-out N] [--cost C --currency USD] [--failed]]": "uso: trilha-spec evidence <task-id> [add --note TEXTO | add --run [--provider P] [--model M] [--tokens-in N] [--tokens-out N] [--cost C --currency USD] [--failed]]",
	"evidence add needs --note, --file, --run, --eval or --attestation":                                                                                                    "evidence add precisa de --note, --file, --run, --eval ou --attestation",
	"note: an unsigned attestation is a claim; it does not count towards a quorum\n":                                                                                       "atenção: uma atestação sem assinatura é uma alegação; não conta para o quórum\n",
	"task %s asks for a review quorum but names no roles":                                                                                                                  "a task %s pede quórum de revisão mas não nomeia papéis",
	"attestation signed with a key the project does not hold: %s":                                                                                                          "atestação assinada com uma chave que o projeto não tem: %s",
	"evidence add --eval needs --metric and --value":                                                                                                                       "evidence add --eval precisa de --metric e --value",
	"--value %q is not a number":        "--value %q não é um número",
	"--threshold %q is not a number":    "--threshold %q não é um número",
	"recorded #%d (%s)\n":               "gravado #%d (%s)\n",
	"recorded #%d (%s), signed by %s\n": "gravado #%d (%s), assinado por %s\n",
	"--key-id needs --sign-key":         "--key-id precisa de --sign-key",
	"%d record(s), %d key(s) in %s\n":   "%d registro(s), %d chave(s) em %s\n",
	"%d invalid signature(s)":           "%d assinatura(s) inválida(s)",
	"usage: trilha-spec keygen <key-id> [--out DIR]   (key-id: lowercase words joined by - or .)": "uso: trilha-spec keygen <key-id> [--out DIR]   (key-id: palavras minúsculas unidas por - ou .)",
	"%s exists; pick another key id":                                               "%s existe; escolha outro key id",
	"private key %s (keep it out of the repository)\npublic key  %s (commit it)\n": "chave privada %s (mantenha fora do repositório)\nchave pública %s (commite)\n",
	// mcp (stderr)
	"trilha-spec mcp %s · %s\ntools: %s\n":                                              "trilha-spec mcp %s · %s\nferramentas: %s\n",
	"read-only; pass --write to offer trilha_move, trilha_evidence and trilha_verify\n": "somente leitura; passe --write para oferecer trilha_move, trilha_evidence e trilha_verify\n",
	// doctor
	"private key %s is inside .trilha; move it out (only `.pub` files belong in keys/)": "a chave privada %s está dentro de .trilha; tire-a de lá (só arquivos `.pub` pertencem a keys/)",
	"✓ %s is healthy\n": "✓ %s está saudável\n",
	"%d problem(s)":     "%d problema(s)",
	"%d warning(s)\n":   "%d aviso(s)\n",
	"spec %s is approved but declares no security impact (assets, trust_boundaries, controls, evidence)": "a spec %s está approved mas não declara impacto de segurança (assets, trust_boundaries, controls, evidence)",
	"missing directory %s (run `trilha-spec init`)":                                                      "falta o diretório %s (rode `trilha-spec init`)",
	"missing %s (run `trilha-spec init`)":                                                                "falta %s (rode `trilha-spec init`)",
	"spec reference does not exist: %s":                                                                  "referência a spec inexistente: %s",
	"spec %s is superseded but no spec names it in `supersedes`":                                         "a spec %s está superseded mas nenhuma spec a cita em `supersedes`",
	"requirement declared by more than one spec: %s":                                                     "requisito declarado por mais de uma spec: %s",
	"task covers a requirement no spec declares: %s":                                                     "a task cobre um requisito que nenhuma spec declara: %s",
	"requirement no task covers: %s":                                                                     "requisito que nenhuma task cobre: %s",
	"milestone project.md does not declare: %s":                                                          "marco que o project.md não declara: %s",
	"milestone %s has no task":                                                                           "o marco %s não tem task",
	"task past its milestone's due date: %s":                                                             "task com o prazo do marco vencido: %s",
	"acceptance metric with no eval evidence: %s":                                                        "métrica de aceite sem evidência eval: %s",
	".trilha/.gitignore ignores everything (`*`): specs, tasks and evidence will not be committed; run `trilha-spec init` to rewrite it": ".trilha/.gitignore ignora tudo (`*`): specs, tasks e evidência não serão commitadas; rode `trilha-spec init` para reescrevê-lo",
}
