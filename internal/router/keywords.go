package router

// Comprehensive Portuguese keywords for routing (Expanded to ~1000+ total)

// Web Search Required keywords
var webSearchKeywords = []string{
	// News & Journalism
	"notícias", "jornal", "manchete", "reportagem", "cobertura", "jornalismo", "imprensa", "mídia",
	"veículo", "publicação", "edição", "jornalista", "repórter", "matéria", "artigo", "texto jornalístico",
	"breaking news", "notícia de última hora", "notícia urgente", "notícia importante",
	"últimas notícias", "notícias recentes", "notícias de hoje", "noticiário", "boletim", "informativo",
	"plantão", "extra", "furo de reportagem", "exclusivo", "em primeira mão", "furo jornalístico",
	"coluna", "editorial", "opinião pública", "repercussão", "escândalo", "polêmica", "denúncia",
	"investigação jornalística", "coletiva de imprensa", "entrevista coletiva", "comunicado oficial",
	"nota oficial", "pronunciamento", "discurso", "declaração", "anúncio", "revelação", "vazamento",
	"fontes", "bastidores", "fofoca", "celebridades", "famosos", "mundo dos famosos", "entretenimento",
	"variedades", "cultura pop", "viral", "trending", "tendências", "assuntos do momento", "tópicos quentes",
	"hashtag", "twitter", "facebook", "instagram", "tiktok", "youtube", "rede social", "mídias sociais",

	// Time-sensitive & Real-time
	"hoje", "agora", "atual", "recente", "último", "mais recente", "atualmente", "neste momento",
	"agora mesmo", "neste instante", "neste exato momento", "atualidade", "atualização",
	"em tempo real", "ao vivo", "live", "transmissão ao vivo", "minuto a minuto", "tempo real",
	"nesta semana", "neste mês", "neste ano", "ontem", "anteontem", "amanhã", "depois de amanhã",
	"fim de semana", "feriado", "próximo", "futuro", "previsão", "tendência", "expectativa",
	"agendado", "programado", "marcado", "adiado", "cancelado", "confirmado", "rumor", "especulação",
	"novidade", "lançamento", "estreia", "inauguração", "abertura", "encerramento", "fechamento",

	// Prices, Finance & Economy
	"cotação", "preço", "valor", "dólar", "euro", "bitcoin", "ações", "bolsa", "ibovespa",
	"mercado financeiro", "câmbio", "taxa de câmbio", "moeda", "criptomoeda", "investimento",
	"papéis", "renda variável", "renda fixa", "selic", "cdi", "ipca", "inflação",
	"taxa selic", "taxa básica", "juros", "rentabilidade", "lucro", "prejuízo", "patrimônio", "fortuna",
	"preço atual", "dólar hoje", "euro hoje", "bitcoin hoje", "ethereum", "litecoin", "blockchain",
	"nft", "defi", "fintech", "banco", "corretora", "carteira", "fundo de investimento", "tesouro direto",
	"poupança", "lci", "lca", "cdb", "debentures", "dividendos", "proventos", "fii", "fundos imobiliários",
	"imposto de renda", "receita federal", "leão", "malha fina", "restituição", "darf", "boleto",
	"fatura", "cartão de crédito", "crédito", "débito", "financiamento", "empréstimo", "hipoteca",
	"consórcio", "seguro", "previdência", "aposentadoria", "inss", "fgts", "pis", "pasep",
	"salário", "remuneração", "bônus", "plr", "dissídio", "sindicato", "greve", "paralisação",
	"economia", "macroeconomia", "microeconomia", "pib", "recessão", "crise", "crescimento",
	"desenvolvimento", "exportação", "importação", "balança comercial", "superávit", "déficit",
	"dívida pública", "orçamento", "teto de gastos", "reforma tributária", "reforma administrativa",

	// Weather & Environment
	"tempo", "clima", "previsão", "chuva", "temperatura", "previsão do tempo", "clima hoje",
	"vai chover", "faz calor", "faz frio", "umidade", "vento", "precipitação", "neve", "granizo",
	"temporal", "tempestade", "chuva forte", "seca", "enchente", "alagamento", "geada", "neblina", "névoa",
	"tempo agora", "meteorologia", "climatempo", "inmet", "cptec", "satélite", "radar", "frente fria",
	"frente quente", "zona de convergência", "el niño", "la niña", "aquecimento global", "mudança climática",
	"efeito estufa", "camada de ozônio", "poluição", "qualidade do ar", "índice uv", "radiação solar",
	"nascer do sol", "pôr do sol", "fase da lua", "maré", "tábua de marés", "ressaca", "ciclone",
	"furacão", "tufão", "tornado", "terremoto", "tsunami", "vulcão", "erupção", "desmatamento",
	"queimada", "incêndio florestal", "amazônia", "pantanal", "cerrado", "mata atlântica", "caatinga",
	"pampa", "bioma", "biodiversidade", "sustentabilidade", "ecologia", "meio ambiente", "preservação",

	// Sports
	"jogo", "partida", "resultado", "placar", "campeonato", "liga", "time", "equipe", "atleta", "jogador",
	"esporte", "futebol", "basquete", "vôlei", "tênis", "natação", "atletismo", "olimpíadas", "mundial",
	"copa", "brasileirão", "libertadores", "copa do brasil", "estadual", "série a", "série b",
	"artilheiro", "gols", "pontos", "vitória", "derrota", "empate", "jogo de hoje", "tabela",
	"classificação", "rodada", "fase de grupos", "mata-mata", "quartas de final", "semifinal", "final",
	"decisão", "título", "taça", "troféu", "medalha", "ouro", "prata", "bronze", "pódio",
	"recorde", "superação", "doping", "lesão", "contratação", "transferência", "mercado da bola",
	"técnico", "treinador", "árbitro", "juiz", "bandeirinha", "var", "pênalti", "falta",
	"cartão amarelo", "cartão vermelho", "expulsão", "substituição", "acréscimos", "prorrogação",
	"pênaltis", "disputa de pênaltis", "torcida", "estádio", "arena", "ingresso", "sócio torcedor",
	"fórmula 1", "f1", "corrida", "gp", "grande prêmio", "pole position", "grid", "largada",
	"chegada", "podium", "piloto", "equipe", "construtores", "box", "pit stop", "ultrapassagem",
	"mma", "ufc", "luta", "combate", "octógono", "cinturão", "nocaute", "finalização", "pesagem",
	"boxe", "judô", "jiu-jitsu", "karatê", "taekwondo", "surfe", "skate", "ginástica",

	// Events, Politics & Society
	"aconteceu", "acontecendo", "ocorreu", "evento", "incidente", "acidente", "tragédia", "catástrofe",
	"desastre", "política", "eleição", "votação", "plebiscito", "referendo", "governo", "presidente",
	"ministro", "deputado", "senador", "prefeito", "governador", "eleito", "candidato", "campanha",
	"votar", "urna", "apuração", "democracia", "ditadura", "golpe", "impeachment", "cassação",
	"corrupção", "lava jato", "pf", "polícia federal", "stf", "supremo tribunal federal", "congresso",
	"câmara", "senado", "planalto", "alvorada", "itamaraty", "esplanada", "manifestação", "protesto",
	"passeata", "greve", "paralisação", "reivindicação", "sindicato", "movimento social", "ong",
	"direitos humanos", "cidadania", "constituição", "lei", "projeto de lei", "medida provisória",
	"decreto", "portaria", "resolução", "julgamento", "sentença", "condenação", "absolvição",
	"prisão", "liberdade", "habeas corpus", "fiança", "delegacia", "boletim de ocorrência", "crime",
	"assalto", "roubo", "furto", "homicídio", "assassinato", "feminicídio", "violência", "segurança",
	"saúde", "educação", "transporte", "habitação", "saneamento", "infraestrutura", "obras",

	// Technology & Science
	"tecnologia", "inovação", "ciência", "descoberta", "pesquisa", "estudo", "artigo científico",
	"nasa", "esa", "spacex", "foguete", "satélite", "missão espacial", "marte", "lua", "universo",
	"galáxia", "estrela", "planeta", "asteroide", "cometa", "buraco negro", "big bang", "física",
	"química", "biologia", "medicina", "vacina", "vírus", "bactéria", "doença", "cura", "tratamento",
	"medicamento", "remédio", "farmácia", "hospital", "médico", "enfermeiro", "paciente", "cirurgia",
	"inteligência artificial", "ia", "ai", "machine learning", "aprendizado de máquina", "deep learning",
	"rede neural", "algoritmo", "robô", "robótica", "automação", "internet", "web", "online",
	"offline", "conexão", "wifi", "5g", "4g", "fibra óptica", "banda larga", "smartphone",
	"celular", "iphone", "android", "app", "aplicativo", "software", "hardware", "computador",
	"notebook", "laptop", "tablet", "gadget", "dispositivo", "eletrônico", "game", "jogo",
	"console", "playstation", "xbox", "nintendo", "pc gamer", "steam", "twitch", "streamer",
}

// Complex Analysis keywords
var complexKeywords = []string{
	// Explanation & Understanding
	"explique", "explicar", "como funciona", "funcionamento", "mecanismo", "processo",
	"como é que", "de que forma", "de que maneira", "como se", "como acontece", "como ocorre",
	"como se dá", "como se processa", "como se desenvolve", "como se forma", "como se cria",
	"como se produz", "como se gera", "elucide", "elucidar", "esclareça", "esclarecer",
	"detalhe", "detalhar", "descreva", "descrever", "demonstre", "demonstrar", "ilustre",
	"ilustrar", "exemplifique", "exemplificar", "contextualize", "contextualizar", "situe",
	"situar", "introduza", "introduzir", "apresente", "apresentar", "exponha", "expor",
	"discorra", "discorrer", "comente", "comentar", "fale sobre", "disserte", "dissertar",
	"resuma", "resumir", "sintetize", "sintetizar", "abrevie", "abreviar", "recapitule",
	"recapitular", "interprete", "interpretar", "traduza", "traduzir", "decodifique", "decodificar",
	"dúvida", "dúvidas", "pergunta", "perguntas", "questão", "questões", "indagação", "indagações",

	// Analysis & Investigation
	"analise", "analisar", "avaliar", "avaliação", "examinar", "exame", "estudar", "estudo",
	"investigar", "investigação", "pesquisar", "pesquisa", "avaliar criticamente",
	"avaliar profundamente", "examinar detalhadamente", "analisar em profundidade",
	"estudar a fundo", "investigar minuciosamente", "dissecar", "esmiuçar", "esquadrinhar",
	"aprofundar", "aprofundamento", "mergulhar", "mergulho", "sondar", "sondagem",
	"explorar", "exploração", "inspecionar", "inspeção", "vistoriar", "vistoria",
	"auditar", "auditoria", "revisar", "revisão", "criticar", "crítica", "julgar", "julgamento",
	"apreciar", "apreciação", "ponderar", "ponderação", "refletir", "reflexão", "meditar",
	"meditação", "contemplar", "contemplação", "observar", "observação", "verificar", "verificação",
	"checar", "checagem", "testar", "teste", "experimentar", "experimento", "experiência",

	// Comparison & Contrast
	"compare", "comparar", "diferença", "diferenças", "semelhança", "semelhanças", "contraste",
	"contrastar", "oposição", "opostos", "similar", "diferente", "distinto", "igual", "equivalente",
	"análogo", "parecido", "comparação", "comparativo", "versus", "vs", "em relação a",
	"em comparação com", "ao contrário de", "diferentemente de", "qual a diferença",
	"distinguir", "distinção", "diferenciar", "diferenciação", "equiparar", "equiparação",
	"assemelhar", "semelhança", "identificar", "identidade", "corresponder", "correspondência",
	"relacionar", "relação", "confrontar", "confronto", "cotejar", "cotejo", "paralelo",
	"fazer um paralelo", "traçar um paralelo", "pontos em comum", "divergências", "convergências",
	"discrepâncias", "disparidades", "analogia", "metáfora", "alegoria", "símile",

	// Why, Reason & Causality
	"por que", "porque", "razão", "motivo", "causa", "causas", "por qual motivo", "por qual razão",
	"qual a razão", "qual o motivo", "qual a causa", "devido a", "em razão de", "em virtude de",
	"em função de", "graças a", "por causa de", "devido ao fato de", "em decorrência de",
	"consequentemente", "portanto", "por isso", "por esse motivo", "causalidade", "efeito",
	"consequência", "impacto", "resultado", "fruto", "produto", "origem", "gênese", "fonte",
	"raiz", "fundamento", "base", "alicerce", "justificativa", "explicação", "pretexto",
	"móvel", "impulso", "estímulo", "incentivo", "motivação", "propósito", "intuito",
	"finalidade", "objetivo", "meta", "alvo", "destino", "fado", "sina", "karma",

	// Pros/Cons & Evaluation
	"vantagens", "desvantagens", "prós", "contras", "pontos positivos", "pontos negativos",
	"benefícios", "malefícios", "lado positivo", "lado negativo", "aspectos positivos",
	"aspectos negativos", "pontos a favor", "pontos contra", "argumentos a favor",
	"argumentos contra", "prós e contras", "vantagens e desvantagens", "pontos fortes",
	"pontos fracos", "qualidades", "defeitos", "virtudes", "vícios", "bônus", "ônus",
	"ganhos", "perdas", "lucros", "prejuízos", "ativos", "passivos", "forças", "fraquezas",
	"oportunidades", "ameaças", "swot", "fofa", "custo-benefício", "trade-off", "compensação",
	"equilíbrio", "balanço", "saldo", "resultado final", "veredicto", "conclusão",

	// History & Context
	"história", "histórico", "origem", "evolução", "desenvolvimento", "passado", "antigamente",
	"no passado", "historicamente", "tradicionalmente", "desde quando", "quando começou",
	"quando surgiu", "como surgiu", "como começou", "cronologia", "linha do tempo", "época",
	"era", "período", "século", "década", "ano", "data histórica", "história de", "biografia",
	"contexto", "contextualização", "cenário", "pano de fundo", "antecedentes", "precedentes",
	"precursores", "pioneiros", "fundadores", "criadores", "inventores", "descobridores",
	"ancestrais", "antepassados", "genealogia", "árvore genealógica", "linhagem", "dinastia",
	"império", "reino", "república", "civilização", "cultura", "sociedade", "humanidade",
	"arqueologia", "antropologia", "sociologia", "filosofia", "política", "economia",

	// Definition & Concept
	"o que é", "defina", "definição", "significado", "conceito", "noção", "ideia", "entender",
	"compreender", "entendimento", "compreensão", "significa", "quer dizer", "o que significa",
	"o que quer dizer", "o que representa", "o que caracteriza", "o que define", "características",
	"atributos", "propriedades", "essência", "natureza", "substância", "cerne", "âmago",
	"núcleo", "centro", "foco", "ponto central", "ideia central", "tema", "tópico", "assunto",
	"matéria", "disciplina", "campo", "área", "domínio", "esfera", "âmbito", "universo",
	"categoria", "classe", "tipo", "gênero", "espécie", "família", "ordem", "grupo",

	// Deep Questions & Relationships
	"qual a relação", "como se relaciona", "qual a conexão", "como se conecta", "qual a influência",
	"como influencia", "qual o impacto", "como impacta", "qual a importância", "por que é importante",
	"qual o papel", "como funciona a relação", "qual a interdependência", "como se interconecta",
	"qual o vínculo", "qual o laço", "qual a ligação", "qual a ponte", "qual a interface",
	"qual a interação", "como interage", "como afeta", "como modifica", "como altera",
	"como transforma", "como muda", "como revoluciona", "como inova", "como melhora",
	"como piora", "quais as consequências", "quais os efeitos", "quais os resultados",
	"quais as implicações", "quais os desdobramentos", "quais as repercussões",

	// Philosophy, Religion & Spirituality
	"deus", "god", "jesus", "cristo", "espírito", "santo", "bíblia", "religião", "fé", "oração", "rezar", "igreja", "templo", "sagrado", "divino", "teologia", "espiritualidade", "alma", "céu", "inferno", "pecado", "perdão", "salvação", "evangelho", "versículo", "salmo", "profeta", "milagre", "bênção", "amém",

	// Emotions & Feelings
	"amor", "amar", "paixão", "ódio", "tristeza", "felicidade", "alegria", "medo", "ansiedade", "depressão", "saudade", "esperança", "confiança", "ciúmes", "inveja", "orgulho", "vergonha", "culpa", "arrependimento", "compaixão", "empatia", "gratidão", "sentimento", "emoção", "psicologia", "mente", "consciência",

	// Natural Sciences (Biology, Physics, Chemistry)
	"biologia", "genética", "evolução", "célula", "dna", "rna", "ecossistema", "física", "mecânica quântica", "relatividade", "termodinâmica", "átomo", "elétron", "química", "tabela periódica", "reação química", "molécula", "orgânica", "inorgânica",

	// Humanities (History, Geography, Sociology)
	"geografia", "geopolítica", "guerra", "revolução", "império", "civilização", "idade média", "renascimento", "iluminismo", "guerra fria", "ditadura", "democracia", "sociologia", "antropologia",
}

// Factual Lookup keywords
var factualKeywords = []string{
	// Questions (Who, What, Where, When)
	"quando", "onde", "quem", "qual", "quantos", "quantas", "qual é", "quem é", "onde fica",
	"quando foi", "quando aconteceu", "quando ocorreu", "quando surgiu", "quando nasceu",
	"quando morreu", "quando criou", "quando inventou", "quando descobriu", "em que ano",
	"em que data", "em que época", "em que período", "aonde", "de onde", "para onde",
	"por onde", "com quem", "de quem", "para quem", "por quem", "o qual", "a qual",
	"os quais", "as quais", "cujo", "cuja", "cujos", "cujas", "quanto", "quanta",

	// Facts & Data
	"data", "ano", "mês", "dia", "local", "lugar", "localização", "país", "cidade", "estado",
	"região", "continente", "capital", "população", "habitantes", "área", "território", "fronteira",
	"limite", "coordenadas", "latitude", "longitude", "altitude", "clima", "idioma", "moeda",
	"religião", "cultura", "gentílico", "naturalidade", "nacionalidade", "etnia", "raça",
	"cor", "gênero", "sexo", "idade", "altura", "peso", "medidas", "tamanho", "dimensões",
	"volume", "capacidade", "velocidade", "distância", "tempo", "duração", "prazo",
	"validade", "vencimento", "garantia", "fabricação", "produção", "origem", "destino",

	// Definitions & Quick Info
	"significado", "definição", "o que é", "o que significa", "conceito", "noção", "ideia básica",
	"resumo", "resumidamente", "em poucas palavras", "de forma resumida", "brevemente",
	"sucintamente", "de maneira concisa", "sinopse", "sumário", "índice", "glossário",
	"vocabulário", "dicionário", "enciclopédia", "wikipédia", "wiki", "verbete", "termo",
	"expressão", "gíria", "ditado", "provérbio", "frase", "citação", "autor", "fonte",
	"referência", "bibliografia", "créditos", "autoria", "copyright", "direitos autorais",

	// Specific Information
	"informação", "dados", "estatísticas", "números", "valores", "características", "atributos",
	"propriedades", "especificações", "detalhes básicos", "informações básicas", "dados básicos",
	"fatos", "curiosidades", "informações gerais", "ficha técnica", "perfil", "biografia",
	"currículo", "histórico", "registro", "documento", "certidão", "diploma", "certificado",
	"comprovante", "recibo", "nota fiscal", "contrato", "acordo", "tratado", "lei",
	"norma", "regra", "regulamento", "estatuto", "código", "manual", "guia", "tutorial",
}

// Mathematical/Calculation keywords
var mathematicalKeywords = []string{
	// Calculation & Operations
	"calcule", "calcular", "conversão", "converter", "quanto é", "quanto dá", "qual o resultado",
	"resultado", "soma", "somar", "subtração", "subtrair", "multiplicação", "multiplicar",
	"divisão", "dividir", "potência", "raiz", "porcentagem", "percentual", "por cento", "%",
	"calcule o valor", "qual o valor", "quanto custa", "quanto vale", "quanto representa",
	"mais", "menos", "vezes", "dividido por", "elevado a", "raiz quadrada", "raiz cúbica",
	"logaritmo", "seno", "cosseno", "tangente", "fatorial", "derivada", "integral", "limite",
	"matriz", "vetor", "determinante", "sistema linear", "equação diferencial", "polinômio",
	"fração", "número misto", "dízima periódica", "número primo", "mdc", "mmc", "divisores",
	"múltiplos", "resto", "quociente", "dividendo", "divisor", "parcela", "fator", "produto",

	// Units & Measurements
	"quilômetro", "km", "metro", "m", "centímetro", "cm", "milímetro", "mm", "quilograma", "kg",
	"grama", "g", "litro", "l", "mililitro", "ml", "graus", "celsius", "fahrenheit", "real",
	"reais", "dólar", "dólares", "euro", "euros", "libra", "peso", "converter para",
	"equivalente a", "corresponde a", "equivale a", "transformar em", "passar para",
	"polegada", "pé", "jarda", "milha", "légua", "onça", "libra", "arroba", "hectare",
	"alqueire", "acre", "galão", "barril", "pint", "quart", "kelvin", "rankine",
	"segundo", "minuto", "hora", "dia", "semana", "mês", "ano", "década", "século", "milênio",
	"joule", "caloria", "watt", "cavalo-vapor", "hp", "volt", "ampere", "ohm", "hertz",
	"byte", "kilobyte", "megabyte", "gigabyte", "terabyte", "petabyte", "bit", "pixel",

	// Math Terms & Areas
	"equação", "fórmula", "cálculo", "matemática", "aritmética", "álgebra", "geometria",
	"trigonometria", "estatística", "probabilidade", "média", "mediana", "moda", "desvio padrão",
	"variância", "regra de três", "proporção", "razão", "fração", "decimal", "inteiro", "número", "números",
	"teorema", "axioma", "postulado", "hipótese", "conjectura", "lema", "corolário", "prova",
	"demonstração", "lógica", "conjunto", "subconjunto", "união", "interseção", "diferença",
	"complementar", "função", "domínio", "imagem", "contradomínio", "gráfico", "eixo",
	"abscissa", "ordenada", "plano cartesiano", "polígono", "triângulo", "quadrado", "retângulo",
	"círculo", "circunferência", "elipse", "parábola", "hipérbole", "esfera", "cubo", "cilindro",
	"cone", "pirâmide", "prisma", "poliedro", "ângulo", "grau", "radiano", "pi",

	// Financial Math
	"juros simples", "juros compostos", "montante", "capital", "taxa", "tempo", "período",
	"amortização", "tabela price", "tabela sac", "vpl", "tir", "payback", "fluxo de caixa",
	"desconto", "abatimento", "acréscimo", "multa", "mora", "correção monetária", "indexador",

	// Advanced Math
	"cálculo", "derivada", "integral", "limite", "álgebra linear", "matriz", "vetor", "autovalor", "autovetor", "estatística", "probabilidade", "distribuição normal", "variância", "desvio padrão", "regressão", "teorema", "geometria analítica",
}

// Creative/Open-ended keywords
var creativeKeywords = []string{
	// Suggestions & Recommendations
	"sugira", "sugerir", "recomende", "recomendar", "indique", "indicar", "indicação",
	"recomendação", "sugestão", "dica", "dicas", "conselho", "conselhos", "orientação",
	"orientações", "proponha", "propor", "proposta", "opções", "alternativas", "possibilidades",
	"ideias para", "inspiração para", "roteiro para", "guia para", "manual para", "tutorial para",
	"passo a passo", "como fazer", "diy", "faça você mesmo", "receita", "modo de preparo",
	"lista de", "top 10", "melhores", "piores", "ranking", "seleção", "curadoria",
	"bebida", "bebidas", "drink", "drinks", "coquetel", "coquetéis", "cocktail", "cocktails",
	"drink para", "bebida para", "sugestão de drink", "sugestão de bebida",

	// Opinions & Perspectives
	"opinião", "opiniões", "o que você acha", "o que pensa", "qual sua opinião", "na sua opinião",
	"na sua visão", "do seu ponto de vista", "pensamento", "pensamentos", "visão", "perspectiva",
	"ponto de vista", "achismo", "achou", "acha", "pensa", "pensou", "considera", "considerou",
	"julga", "julgou", "acredita", "acreditou", "crê", "crer", "supõe", "supor", "imagina",
	"imaginar", "sente", "sentir", "percebe", "perceber", "entende", "entender", "interpreta",
	"interpretar", "analisa", "analisar", "avalia", "avaliar", "critica", "criticar",

	// Creative Writing & Generation
	"crie", "criar", "imagine", "imaginar", "invente", "inventar", "desenvolva", "desenvolver",
	"elabore", "elaborar", "construa", "construir", "formule", "formular", "proponha", "propor",
	"criativo", "criatividade", "original", "originalidade", "inovador", "inovação",
	"escreva", "escrever", "redija", "redigir", "componha", "compor", "narre", "narrar",
	"conte", "contar", "relate", "relatar", "descreva", "descrever", "poema", "poesia",
	"verso", "estrofe", "rima", "soneto", "haicai", "crônica", "conto", "fábula", "lenda",
	"mito", "história", "romance", "novela", "roteiro", "script", "diálogo", "monólogo",
	"carta", "email", "mensagem", "texto", "artigo", "ensaio", "resenha", "sinopse",
	"slogan", "tagline", "manchete", "título", "nome", "marca", "logo", "design",

	// Brainstorming & Ideation
	"ideias", "ideia", "possibilidades", "possibilidade", "alternativas", "alternativa", "opções",
	"opção", "escolhas", "escolha", "variantes", "variante", "opções disponíveis", "soluções",
	"solução", "abordagens", "abordagem", "estratégias", "estratégia", "métodos", "método",
	"táticas", "tática", "técnicas", "técnica", "ferramentas", "ferramenta", "recursos",
	"recurso", "meios", "meio", "caminhos", "caminho", "vias", "via", "modos", "modo",
	"maneiras", "maneira", "formas", "forma", "estilos", "estilo", "tipos", "tipo",
	"brainstorm", "brainstorming", "tempestade de ideias", "mapa mental", "fluxograma",
	"esquema", "rascunho", "esboço", "plano", "planejamento", "projeto", "protótipo",

	// Home & Lifestyle
	"casa", "decoração", "design de interiores", "jardinagem", "plantas", "culinária", "receita", "gastronomia", "limpeza", "organização", "diy", "faça você mesmo", "artesanato", "moda", "estilo", "beleza", "maquiagem", "skincare",
}

// Negative keywords to filter false positives
// If any of these match, the category score is penalized or zeroed
var negativeKeywords = map[string][]string{
	"web_search": {
		"crie", "imagine", "invente", "escreva", "redija", "traduza", "explique", "defina",
		"o que é", "significado", "conceito", "resuma", "sintetize", "analise", "compare",
	},
	"factual": {
		"crie", "imagine", "invente", "sugira", "recomende", "opinião",
	},
}
