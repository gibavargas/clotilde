package router

func init() {
	complexKeywords = append(complexKeywords,
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
		"budismo", "hinduísmo", "islamismo", "judaísmo", "espiritismo", "candomblé", "umbanda", "ateísmo", "agnosticismo",
		"meditação", "yoga", "karma", "dharma", "nirvana", "reencarnação", "ressurreição", "iluminação", "consciência",
		"ética", "moral", "metafísica", "epistemologia", "lógica", "estética", "existencialismo", "niilismo", "estoicismo",
		"platão", "aristóteles", "sócrates", "kant", "nietzsche", "marx", "freud", "jung", "dalai lama",

		// Emotions & Feelings
		"amor", "amar", "paixão", "ódio", "tristeza", "felicidade", "alegria", "medo", "ansiedade", "depressão", "saudade", "esperança", "confiança", "ciúmes", "inveja", "orgulho", "vergonha", "culpa", "arrependimento", "compaixão", "empatia", "gratidão", "sentimento", "emoção", "psicologia", "mente", "consciência",
		"autoestima", "autoconhecimento", "inteligência emocional", "resiliência", "estresse", "burnout", "terapia", "psicanálise", "trauma", "fobia", "pânico", "luto", "superação", "motivação", "inspiração", "propósito", "felicidade", "bem-estar", "saúde mental",

		// Natural Sciences (Biology, Physics, Chemistry)
		"biologia", "genética", "evolução", "célula", "dna", "rna", "ecossistema", "física", "mecânica quântica", "relatividade", "termodinâmica", "átomo", "elétron", "química", "tabela periódica", "reação química", "molécula", "orgânica", "inorgânica",
		"fotossíntese", "respiração celular", "mitose", "meiose", "proteína", "enzima", "carboidrato", "lipídio", "vitamina",
		"mamífero", "ave", "réptil", "anfíbio", "peixe", "inseto", "aracnídeo", "crustáceo", "molusco", "planta", "flor", "fruto",
		"gravidade", "força", "energia", "trabalho", "potência", "velocidade", "aceleração", "massa", "peso", "densidade",
		"eletricidade", "magnetismo", "luz", "som", "onda", "frequência", "comprimento de onda", "espectro", "laser",
		"ácido", "base", "sal", "óxido", "ph", "solução", "mistura", "substância", "elemento", "composto", "ligação química",

		// Humanities (History, Geography, Sociology)
		"geografia", "geopolítica", "guerra", "revolução", "império", "civilização", "idade média", "renascimento", "iluminismo", "guerra fria", "ditadura", "democracia", "sociologia", "antropologia",
		"capitalismo", "socialismo", "comunismo", "liberalismo", "conservadorismo", "fascismo", "nazismo", "anarquismo",
		"globalização", "migração", "imigração", "emigração", "refugiado", "fronteira", "território", "nação", "estado",
		"cultura", "identidade", "gênero", "raça", "etnia", "classe social", "desigualdade", "pobreza", "riqueza",
		"urbanização", "industrialização", "agricultura", "pecuária", "extrativismo", "sustentabilidade", "meio ambiente",
	)
}
