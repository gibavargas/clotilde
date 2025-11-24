# Correção de Segurança: Autenticação Cloud Run

## Problema Identificado

O serviço está configurado com **"Permitir sem autenticação"** no Cloud Run, o que significa:
- ✅ Qualquer pessoa na internet pode acessar o endpoint
- ⚠️ Expõe o serviço a ataques de força bruta
- ⚠️ Permite tentativas de descobrir o API key
- ⚠️ Aumenta risco de DDoS

## Proteções Atuais

Mesmo com acesso público, temos:
- ✅ Autenticação por API key na aplicação (X-API-Key header)
- ✅ Rate limiting (10 req/min, 100 req/hour)
- ✅ Input validation
- ✅ Logging seguro

## Soluções

### Opção 1: Manter Público (Recomendado para Apple Shortcuts)

**Prós:**
- Funciona diretamente com Apple Shortcuts sem configuração adicional
- Mais simples de usar

**Contras:**
- Endpoint exposto publicamente
- Risco de ataques

**Mitigações:**
- Rate limiting já implementado
- API key forte (64 caracteres hex)
- Monitoramento de logs

### Opção 2: Restringir IAM (Mais Seguro)

Requer autenticação do Google Cloud, mas complica o uso com Apple Shortcuts.

### Opção 3: Híbrido (Melhor Segurança)

Manter público mas adicionar:
- Rate limiting mais agressivo
- IP whitelisting (se possível)
- Cloud Armor para proteção DDoS
- Monitoramento e alertas

## Recomendação

Para uso com Apple Shortcuts, **manter público** mas:
1. ✅ Monitorar logs regularmente
2. ✅ Rotacionar API key periodicamente
3. ✅ Configurar alertas para atividade suspeita
4. ✅ Considerar Cloud Armor se houver muitos ataques

O API key na aplicação já protege contra uso não autorizado.

