# Roadmap — Enlighten-Brasil/terraform-provider-dokploy

Fork independente de `ahmedali6/terraform-provider-dokploy` (último sync:
v0.8.0). Mantemos compat com o Dokploy **≥ 0.30.x** (nossa referência:
`manager.bragi.enspace.io`).

Branches: `enspace-v0.8.0` (release atual) — próximas: `enspace-vX.Y`
alinhadas às versões do servidor que adotarmos.

## Fixes aplicados (v0.8.0-enspace.1)

- `application.remove` → `application.delete` (destroy de apps era no-op).
- Delete do env default `production` tolerado (adotado no create).
- Create do postgres preserva `app_name` prefix (inconsistent-result).
- `saveDockerProvider` sempre envia creds (vazio = dockerhub anônimo).

## Features implementadas (v0.8.0-enspace.2)

- `dokploy_domain.enabled` — toggle v0.30.0 (rota sai/entra no Traefik sem
  deletar). Create normaliza: servidor só honra `enabled` em `domain.update`.
- `dokploy_redis.deploy_on_create` / `dokploy_postgres.deploy_on_create`
  (`redis.deploy`/`postgres.deploy`).

## Backlog priorizado (do changelog do Dokploy)

| Prio | Item | Origem | Esforço |
|---|---|---|---|
| P2 | `middlewares` em `dokploy_domain` | v0.29.0 | baixo |
| P2 | `dokploy_libsql` (novo db type) | v0.29.0 | médio |
| P2 | `deploy_on_create` p/ mysql/mariadb/mongo | simétrico | baixo |
| P3 | Vault providers (`${{vault:...}}` em env) | v0.30.0 | avaliar |
| P3 | Networks attach por serviço | v0.30.0 | alto |
| P3 | DNS providers (Cloudflare/Route53) | v0.30.0 | alto |

## Processo

1. Não interagimos mais com upstream; merges de changelog do Dokploy viram
   itens deste roadmap.
2. Feature/fix: commit na branch `enspace-*` da release + build local
   (`go build`) + validação contra `manager.bragi.enspace.io`.
3. Release com binários multiplataforma (`vX.Y.Z-enspace.N`) após validar
   apply/destroy de ponta a ponta.
4. Upgrade do servidor Dokploy segue smoke test do repo `enlighten-infra`
   antes de subir a versão-alvo do fork.
