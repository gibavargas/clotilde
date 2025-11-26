# Guia Completo: Criar Shortcut Clotilde no iPhone

## Passo a Passo Detalhado

### 1. Abrir o App Shortcuts

1. No seu iPhone, encontre o app **"Atalhos"** (ícone roxo com círculos)
2. Toque para abrir
3. Se não tiver o app, baixe na App Store: "Atalhos" (Shortcuts)

### 2. Criar Novo Shortcut

1. No canto inferior direito, toque no botão **"+"** (mais)
2. Toque em **"Adicionar Ação"** no topo

---

## Ação 1: Dictate Text (Ditar Texto)

1. Na barra de busca, digite: **"ditar"** ou **"dictate"**
2. Toque em **"Ditar Texto"**
3. Configure:
   - **Idioma**: Toque em "Idioma" e selecione **"Português (Brasil)"**
   - **Mostrar Confirmação**: Desligue (OFF) para uso mais rápido no CarPlay

---

## Ação 2: Get Contents of URL (Obter Conteúdo de URL)

1. Toque em **"Adicionar Ação"** novamente
2. Na busca, digite: **"URL"** ou **"obter conteúdo"**
3. Toque em **"Obter Conteúdo de URL"**
4. Configure:

### 4.1. Método HTTP
- Toque no campo que diz "GET"
- Selecione **"POST"**

### 4.2. URL
- Toque no campo de URL
- Apague qualquer texto que estiver lá
- Digite a URL do seu serviço (obtenha com: `gcloud run services describe clotilde --region=us-central1 --format="value(status.url)"`)
  ```
  YOUR_SERVICE_URL/chat
  ```

### 4.3. Cabeçalhos (Headers)
- Toque em **"Mostrar Mais"** ou **"Headers"**
- Toque em **"Adicionar Campo"** ou o botão **"+"**
- Adicione **2 cabeçalhos**:

**Cabeçalho 1:**
- **Chave**: `Content-Type`
- **Valor**: `application/json`

**Cabeçalho 2:**
- **Chave**: `X-API-Key`
- **Valor**: `YOUR_API_KEY` (obtenha do Secret Manager: `gcloud secrets versions access latest --secret=clotilde-api-key`)

### 4.4. Corpo da Requisição (Request Body)
- Toque em **"Corpo"** ou **"Request Body"**
- Selecione **"JSON"**
- **ATENÇÃO**: O Shortcuts vai mostrar um formulário com campos "Chave" e "Valor"

**Passo a passo no formulário Chave/Valor:**

1. **Campo "Chave"**: 
   - Digite apenas: `message`
   - (sem aspas, sem dois pontos, apenas a palavra "message")

2. **Campo "Valor"**: 
   - Toque no campo "Valor"
   - Procure pelo botão de **variável** (geralmente um ícone de círculo com texto ou botão "Variável" ao lado do campo)
   - Se não aparecer, toque no campo "Valor" novamente ou procure por um ícone de círculo
   - Toque no botão de variável
   - Selecione **"Texto Ditado"** da lista (é a variável da Ação 1 - "Ditar Texto")
   - O Shortcuts vai criar automaticamente: `{"message": "[Texto Ditado]"}`

**Se aparecer campo de texto JSON (menos comum):**
- Digite: `{"message": ""}`
- Toque dentro das aspas vazias `""`
- Selecione "Texto Ditado"

**Dica importante**: 
- O botão de variável pode aparecer como um círculo com texto dentro ou um botão "Variável"
- Se não encontrar, tente tocar várias vezes no campo "Valor" ou deslize para ver mais opções
- Certifique-se de selecionar "Texto Ditado" da Ação 1, não outro texto

---

## Ação 3: Get Dictionary from Input (Obter Dicionário da Entrada)

1. Toque em **"Adicionar Ação"**
2. Na busca, digite: **"dicionário"** ou **"dictionary"**
3. Toque em **"Obter Dicionário da Entrada"**
4. A entrada deve estar automaticamente conectada à ação anterior

---

## Ação 4: Get Value for Key (Obter Valor da Chave)

1. Toque em **"Adicionar Ação"**
2. Na busca, digite: **"obter valor"** ou **"get value"**
3. Toque em **"Obter Valor da Chave"**
4. Configure:
   - **Chave**: Digite `response` (sem aspas)
   - **Dicionário**: Deve estar automaticamente conectado à ação anterior

---

## Ação 5: Speak Text (Falar Texto)

1. Toque em **"Adicionar Ação"**
2. Na busca, digite: **"falar"** ou **"speak"**
3. Toque em **"Falar Texto"**
4. Configure:
   - **Idioma**: Toque e selecione **"Português (Brasil)"**
   - **Velocidade**: Ajuste para **0.5** (mais lento, melhor para dirigir)
   - **Texto**: Deve estar automaticamente conectado à ação anterior

---

## Configurar o Shortcut

### 1. Nomear o Shortcut

1. No topo da tela, toque no nome do shortcut (provavelmente "Novo Atalho")
2. Digite: **"Clotilde"**
3. Toque em **"Concluído"**

### 2. Configurar para CarPlay

1. Toque nos **três pontinhos** (**...**) no canto superior direito
2. Toque no ícone de **engrenagem** (⚙️) no canto superior direito
3. Ative:
   - ✅ **Mostrar no CarPlay**
   - ✅ **Mostrar no Apple Watch** (opcional)
4. Toque em **"Concluído"**

### 3. Adicionar ao Siri

1. Ainda na tela de configurações do shortcut
2. Toque em **"Adicionar ao Siri"**
3. Toque no botão de gravar
4. Diga: **"Falar com Clotilde"**
5. Toque em **"Concluído"**

---

## Testar o Shortcut

### Teste 1: No iPhone

1. Diga: **"Hey Siri, Falar com Clotilde"**
2. Quando o microfone aparecer, diga: **"Olá, como você está?"**
3. Clotilde deve responder em português

### Teste 2: No CarPlay

1. Conecte o iPhone ao carro
2. No CarPlay, procure pelo ícone do **"Atalhos"** (Shortcuts)
3. Toque em **"Clotilde"**
4. Ou diga: **"Hey Siri, Falar com Clotilde"**

---

## Solução de Problemas

### O shortcut não aparece no CarPlay

1. Verifique se **"Mostrar no CarPlay"** está ativado
2. Reinicie o iPhone
3. Desconecte e reconecte o iPhone ao carro
4. Verifique se o CarPlay está atualizado

### Erro "Invalid API key"

1. Verifique se o cabeçalho `X-API-Key` está correto:
   ```
   YOUR_API_KEY (obtenha do Secret Manager)
   ```
2. Verifique se não há espaços extras
3. Verifique se está usando POST, não GET

### Erro "Rate limit exceeded"

- Você fez muitas requisições
- Aguarde alguns minutos e tente novamente
- Limite: 10 requisições/minuto, 100/hora

### Clotilde não responde

1. Verifique sua conexão com internet
2. Verifique se a URL está correta (obtenha com: `gcloud run services describe clotilde --region=us-central1 --format="value(status.url)"`)
   ```
   YOUR_SERVICE_URL/chat
   ```
3. Teste acessando a URL no navegador (deve dar erro, mas confirma que está online)

---

## Valores Importantes (Copiar e Colar)

### URL do Serviço:
```
YOUR_SERVICE_URL/chat
```

**Como obter**: Execute `gcloud run services describe clotilde --region=us-central1 --format="value(status.url)"`

### API Key:
```
YOUR_API_KEY
```

**Como obter**: Execute `gcloud secrets versions access latest --secret=clotilde-api-key --project=your-project-id`

### Cabeçalhos:
- **Content-Type**: `application/json`
- **X-API-Key**: `YOUR_API_KEY` (obtenha do Secret Manager)

### Corpo JSON:
```json
{
  "message": "[Texto Ditado]"
```
(Substitua `[Texto Ditado]` pela variável da ação "Ditar Texto")

---

## Dicas

1. **Teste primeiro no iPhone** antes de usar no carro
2. **Fale claramente** quando ditar
3. **Aguarde a resposta** - pode levar 2-5 segundos
4. **Use frases curtas** - Clotilde responde melhor a perguntas diretas
5. **Exemplos de perguntas**:
   - "Qual é a temperatura hoje?"
   - "Como está o trânsito?"
   - "Toca uma música"
   - "Qual é a previsão do tempo?"

---

## Pronto!

Agora você tem o Clotilde configurado e pronto para usar no CarPlay! 🚗

Se tiver problemas, verifique a seção "Solução de Problemas" acima.

