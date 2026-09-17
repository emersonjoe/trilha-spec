# Spec 008 — Evidência assinada

- **Issue**: [#7](https://github.com/emersonjoe/trilha-spec/issues/7) — a issue é a fonte do escopo.
- **Task**: `TASK-008` em `.trilha/tasks/`.
- **Versão**: 0.2 do protocolo (§1, §5 e §8); campo aditivo na evidência.

## Por quê

Um registro de evidência era um arquivo JSON que qualquer um podia escrever à mão. O
revisor — e o control plane — não distinguia o que um runner atestou do que alguém digitou,
e um registro editado depois do fato parecia igual ao original. Sem isso, "done quando a
evidência existe" vale só enquanto todos são honestos.

## O que muda

- `task.Evidence` ganha `signature{alg, key_id, sig}` (opcional, `omitempty`). `alg` é só
  `ed25519`; `key_id` é `^[a-z0-9]+([.-][a-z0-9]+)*$`; `sig` é base64 padrão.
- **Forma canônica**: o registro sem `signature`, chaves ordenadas, JSON compacto, sem escape
  de HTML (`task.Canonical`). É isso que se assina e que se confere.
- **Chaves**: públicas em `.trilha/keys/<key_id>.pub` (PEM PKIX, commitadas); privadas em
  PEM PKCS8 fora de `.trilha` — `.gitignore` ganha `keys/*.key`; `doctor` acusa
  `private-key-in-repo`.
- **Vereditos**: `unsigned`, `valid`, `invalid` (`task.Keyring.Check`/`CheckAll`,
  `task.Checked{Evidence, verdict, reason}`).
- CLI: `keygen <key-id> [--out DIR]` (privada em `DIR/<id>.key` 0600, padrão
  `~/.trilha/keys`; pública em `.trilha/keys/<id>.pub`); `evidence add … --sign-key ARQUIVO
  [--key-id ID]`; `evidence <task> --verify [--keys DIR]` (coluna de veredito, falha com
  assinatura inválida; `--json` expõe `verdict`/`reason`).
- MCP: `trilha_list_evidence` (leitura) responde `CheckAll` contra `.trilha/keys`.
- Pacote de contexto: cada registro marcado `· signed by K`, `· SIGNATURE INVALID (motivo)`
  ou `· unverified`; o `--json` do pacote carrega `verdict`.

```bash
trilha-spec keygen runner-01
trilha-spec evidence TASK-001 add --run --by runner-01 --model claude-sonnet-5 \
  --sign-key ~/.trilha/keys/runner-01.key
trilha-spec evidence TASK-001 --verify
```

## Fora de escopo

- Outros algoritmos, rotação e revogação de chave, cadeia de confiança (control plane).
- Assinar `verify` automaticamente (o runner decide quando assina).

## Constitution Check

| Princípio | Como respeita |
|---|---|
| I — o protocolo é o produto | §5 define forma canônica e vereditos antes do código |
| II — Markdown + front matter / JSON | `signature` é campo do JSON; chaves são PEM em `keys/`, só `.pub` |
| III — zero dependências | `crypto/ed25519`, `crypto/x509`, `encoding/pem` da stdlib |
| V — teste primeiro | `TestSignAndVerifyEvidence`, e2e `TestSignedEvidenceCLI` |

## Tarefas

- [x] T001 Testes: assinar, conferir, registro editado, chave desconhecida, `doctor`
- [x] T002 `task/sign.go`, `RecordSigned`, `Layout.Keys`, gitignore, `ProblemPrivateKey`
- [x] T003 CLI `keygen`/`--sign-key`/`--verify`, i18n, MCP `trilha_list_evidence`, pacote
- [x] T004 Protocolo §1/§5/§8 (en, pt-BR), README (en, pt-BR), capítulo hands-on nas duas locales
- [x] T005 `make test` verde; TASK-008 fechada com evidência assinada por `claude-opus-5`

## Aceitação

- **SC-001** `evidence <task> --verify` responde `valid` para um registro assinado com chave em `.trilha/keys`.
- **SC-002** O mesmo registro, editado, responde `invalid` e o comando falha.
- **SC-003** Registro sem assinatura responde `unsigned`, nunca `invalid`.
- **SC-004** `doctor` acusa um `.key` dentro de `.trilha/keys`.
